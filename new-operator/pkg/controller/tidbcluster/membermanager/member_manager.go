// Copyright 2018 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// See the License for the specific language governing permissions and
// limitations under the License.

package membermanager

import "github.com/pingcap/tidb-operator/new-operator/pkg/apis/pingcap.com/v1"

// MemberManager implements the logic for syncing all TidbCluster members.
type MemberManager interface {
	// Sync	implements the logic for syncing all TidbCluster members.
	// Implements can only modify the status of TidbCluster.
	Sync(*v1.TidbCluster) error
}