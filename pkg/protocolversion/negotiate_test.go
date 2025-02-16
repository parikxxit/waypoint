// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package protocolversion

import (
	"testing"

	"github.com/stretchr/testify/require"

	pb "github.com/hashicorp/waypoint/pkg/server/gen"
)

func TestNegotiate(t *testing.T) {
	cases := []struct {
		Name           string
		Client, Server *pb.VersionInfo_ProtocolVersion
		Err            string
		Vsn            uint32
	}{
		{
			"latest",
			&pb.VersionInfo_ProtocolVersion{Minimum: 1, Current: 10},
			&pb.VersionInfo_ProtocolVersion{Minimum: 1, Current: 10},
			"",
			10,
		},

		{
			"client older but can negotiate",
			&pb.VersionInfo_ProtocolVersion{Minimum: 1, Current: 6},
			&pb.VersionInfo_ProtocolVersion{Minimum: 4, Current: 10},
			"",
			6,
		},

		{
			"client older but can negotiate minimum",
			&pb.VersionInfo_ProtocolVersion{Minimum: 1, Current: 4},
			&pb.VersionInfo_ProtocolVersion{Minimum: 4, Current: 10},
			"",
			4,
		},

		{
			"server older but can negotiate",
			&pb.VersionInfo_ProtocolVersion{Minimum: 4, Current: 10},
			&pb.VersionInfo_ProtocolVersion{Minimum: 1, Current: 5},
			"",
			5,
		},

		{
			"server older but can negotiate minimum",
			&pb.VersionInfo_ProtocolVersion{Minimum: 4, Current: 10},
			&pb.VersionInfo_ProtocolVersion{Minimum: 1, Current: 4},
			"",
			4,
		},

		{
			"client too old",
			&pb.VersionInfo_ProtocolVersion{Minimum: 1, Current: 4},
			&pb.VersionInfo_ProtocolVersion{Minimum: 5, Current: 6},
			ErrClientOutdated.Error(),
			0,
		},

		{
			"server too old",
			&pb.VersionInfo_ProtocolVersion{Minimum: 5, Current: 6},
			&pb.VersionInfo_ProtocolVersion{Minimum: 1, Current: 4},
			ErrServerOutdated.Error(),
			0,
		},
	}

	for _, tt := range cases {
		t.Run(tt.Name, func(t *testing.T) {
			require := require.New(t)

			vsn, err := Negotiate(tt.Client, tt.Server)
			if tt.Err != "" {
				require.Error(err)
				require.Contains(err.Error(), tt.Err)
				return
			}

			require.NoError(err)
			require.Equal(vsn, tt.Vsn)
		})
	}
}
