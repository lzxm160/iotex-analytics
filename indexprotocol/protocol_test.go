// Copyright (c) 2020 IoTeX
// This is an alpha (internal) release and is not suitable for production. This source code is provided 'as is' and no
// warranties are given as to title or non-infringement, merchantability or fitness for purpose and, to the extent
// permitted by law, all liability for your use of the code is disclaimed. This source code is governed by Apache
// License 2.0 that can be found in the LICENSE file.

package indexprotocol

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

const candidateName = "63727970746f6c696f6e7378"

func TestEnDecodeName(t *testing.T) {
	require := require.New(t)
	decoded, err := DecodeDelegateName(candidateName)
	require.NoError(err)
	fmt.Println(decoded)

	encoded, err := EncodeDelegateName(decoded)
	require.NoError(err)
	require.Equal(candidateName, encoded)
}
