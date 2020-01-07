# SOPS Test Data

This directory contains everything required to run the unit-tests for the [sops](https://github.com/mozilla/sops) integration.

For the unit-tests we use a PGP master key, which has been generated using the following command:

```
$ GNUPGHOME=.gnupg gpg --batch --gen-key gpg-gen-key-data.txt
```

It requires the [gpg](https://gnupg.org/) tool, and uses the data from the [gpg-gen-key-data.txt](gpg-gen-key-data.txt) file to create a key. This command should be run from inside this directory (`testdata`), and will write files inside the `.gnupg` directory.
Note that the key doesn't have any passphrase, to make it easier to use in a unit-testing environment.

The key fingerprint was then retrieved using the following command:

```
$ GNUPGHOME=.gnupg gpg --list-secret-keys --keyid-format LONG
```

In an output such as

```
sec   rsa2048/9B9FE83AF8A3B7CB 2020-01-03 [SCEA]
      743D655795120B297F0493279B9FE83AF8A3B7CB
uid                [  ultime ] scribe <scribe@example.com>
ssb   rsa2048/220F343C57C5E3A4 2020-01-03 [SEA]
```

The fingerprint we care about is `9B9FE83AF8A3B7CB` - this is the part we need to put in the [sops_test.go](sops_test.go) file to identify the master key to use to encrypt/decrypt files using sops.

All those operations have already been done once, and you should not need to do anything unless you want to regenerate a new key for some reason.
