package sops

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"go.mozilla.org/sops/v3"
	"go.mozilla.org/sops/v3/aes"
	"go.mozilla.org/sops/v3/cmd/sops/common"
	"go.mozilla.org/sops/v3/cmd/sops/formats"
	"go.mozilla.org/sops/v3/keyservice"

	"github.com/dailymotion/octopilot/update/value"
)

type SopsUpdater struct {
	FilePath string
	Format   formats.Format
	Key      string
	Store    sops.Store
	Valuer   value.Valuer
}

func NewUpdater(params map[string]string, valuer value.Valuer) (*SopsUpdater, error) {
	updater := &SopsUpdater{}

	updater.FilePath = params["file"]
	if len(updater.FilePath) == 0 {
		return nil, errors.New("missing file parameter")
	}

	updater.Key = params["key"]
	if len(updater.Key) == 0 {
		return nil, errors.New("missing key parameter")
	}

	updater.Format = formats.FormatForPathOrString(updater.FilePath, params["format"])
	updater.Store = common.StoreForFormat(updater.Format)
	updater.Valuer = valuer

	return updater, nil
}

func (u SopsUpdater) Update(ctx context.Context, repoPath string) (bool, error) {
	var (
		filePath = filepath.Join(repoPath, u.FilePath)
		cipher   = aes.NewCipher()
		svcs     = []keyservice.KeyServiceClient{keyservice.NewLocalClient()}
	)

	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return false, fmt.Errorf("failed to access file %s: %w", u.FilePath, err)
	}

	tree, err := common.LoadEncryptedFileWithBugFixes(common.GenericDecryptOpts{
		Cipher:      cipher,
		InputStore:  u.Store,
		InputPath:   filePath,
		KeyServices: svcs,
	})
	if err != nil {
		return false, fmt.Errorf("failed to load encrypted file %s: %w", u.FilePath, err)
	}

	dataKey, err := common.DecryptTree(common.DecryptTreeOpts{
		Cipher:      cipher,
		Tree:        tree,
		KeyServices: svcs,
	})
	if err != nil {
		return false, fmt.Errorf("failed to decrypt tree for %s: %w", u.FilePath, err)
	}

	originalData, err := u.Store.EmitPlainFile(tree.Branches)
	if err != nil {
		return false, fmt.Errorf("failed to emit original tree for %s: %w", u.FilePath, err)
	}

	path := convertKeyToPath(u.Key)
	value, err := u.Valuer.Value(ctx, repoPath)
	if err != nil {
		return false, fmt.Errorf("failed to get value: %w", err)
	}
	for i := range tree.Branches {
		// FIXME if the path top-level element doesn't exist, it will return a new branch with only our path
		// and so the existing other top-level elements will be lost
		tree.Branches[i] = tree.Branches[i].Set(path, value)
	}

	// check if we updated something or not, before re-encrypting...
	updatedData, err := u.Store.EmitPlainFile(tree.Branches)
	if err != nil {
		return false, fmt.Errorf("failed to emit updated tree for %s: %w", u.FilePath, err)
	}
	if string(updatedData) == string(originalData) {
		return false, nil
	}

	err = common.EncryptTree(common.EncryptTreeOpts{
		DataKey: dataKey,
		Tree:    tree,
		Cipher:  cipher,
	})
	if err != nil {
		return false, fmt.Errorf("failed to encrypt tree for %s: %w", u.FilePath, err)
	}

	encryptedFile, err := u.Store.EmitEncryptedFile(*tree)
	if err != nil {
		return false, fmt.Errorf("failed to generate re-encrypted file %s: %w", u.FilePath, err)
	}

	err = ioutil.WriteFile(filePath, encryptedFile, fileInfo.Mode())
	if err != nil {
		return false, fmt.Errorf("failed to write re-encrypted data to file %s: %w", u.FilePath, err)
	}

	return true, nil
}

func (u SopsUpdater) Message() (title, body string) {
	title = fmt.Sprintf("Update %s %s", u.FilePath, u.Key)
	body = fmt.Sprintf("Updating sops-encrypted file `%s` key `%s`", u.FilePath, u.Key)
	return title, body
}

func (u SopsUpdater) String() string {
	return fmt.Sprintf("Sops[key=%s,file=%s]", u.Key, u.FilePath)
}

func convertKeyToPath(key string) []interface{} {
	path := make([]interface{}, 0)
	for _, entry := range strings.Split(key, ".") {
		path = append(path, entry)
	}
	return path
}
