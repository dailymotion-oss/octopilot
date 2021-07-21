package value

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStringValuerValue(t *testing.T) {
	t.Parallel()

	expected := "some value"
	valuer := StringValuer(expected)

	actual, err := valuer.Value(context.Background(), ".")
	require.NoError(t, err)
	assert.Equal(t, expected, actual)
}
