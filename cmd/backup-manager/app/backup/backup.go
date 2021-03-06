// Copyright 2019 PingCAP, Inc.
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

package backup

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"path"

	"github.com/gogo/protobuf/proto"
	kvbackup "github.com/pingcap/kvproto/pkg/backup"
	"github.com/pingcap/tidb-operator/cmd/backup-manager/app/constants"
	"github.com/pingcap/tidb-operator/cmd/backup-manager/app/util"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/klog"
)

// Options contains the input arguments to the backup command
type Options struct {
	util.GenericOptions
}

func (bo *Options) backupData(backup *v1alpha1.Backup) (string, error) {
	clusterNamespace := backup.Spec.BR.ClusterNamespace
	if backup.Spec.BR.ClusterNamespace == "" {
		clusterNamespace = backup.Namespace
	}
	args, remotePath, err := constructOptions(backup)
	if err != nil {
		return "", err
	}
	args = append(args, fmt.Sprintf("--pd=%s-pd.%s:2379", backup.Spec.BR.Cluster, clusterNamespace))
	if backup.Spec.BR.EnableTLSClient {
		args = append(args, fmt.Sprintf("--ca=%s", constants.ServiceAccountCAPath))
		args = append(args, fmt.Sprintf("--cert=%s", path.Join(constants.BRCertPath, corev1.TLSCertKey)))
		args = append(args, fmt.Sprintf("--key=%s", path.Join(constants.BRCertPath, corev1.TLSPrivateKeyKey)))
	}

	var btype string
	if backup.Spec.Type == "" {
		btype = string(v1alpha1.BackupTypeFull)
	} else {
		btype = string(backup.Spec.Type)
	}
	fullArgs := []string{
		"backup",
		btype,
	}
	fullArgs = append(fullArgs, args...)
	klog.Infof("Running br command with args: %v", fullArgs)
	output, err := exec.Command("br", fullArgs...).CombinedOutput()
	if err != nil {
		return remotePath, fmt.Errorf("cluster %s, execute br command %v failed, output: %s, err: %v", bo, fullArgs, string(output), err)
	}
	klog.Infof("Backup data for cluster %s successfully, output: %s", bo, string(output))
	return remotePath, nil
}

// getCommitTs get backup position from `EndVersion` in BR backup meta
func getCommitTs(backup *v1alpha1.Backup) (uint64, error) {
	var commitTs uint64
	s, err := util.NewRemoteStorage(backup)
	if err != nil {
		return commitTs, err
	}
	defer s.Close()
	ctx := context.Background()
	exist, err := s.Exists(ctx, constants.MetaFile)
	if err != nil {
		return commitTs, err
	}
	if !exist {
		return commitTs, fmt.Errorf("%s not exist", constants.MetaFile)

	}
	metaData, err := s.ReadAll(ctx, constants.MetaFile)
	if err != nil {
		return commitTs, err
	}
	backupMeta := &kvbackup.BackupMeta{}
	err = proto.Unmarshal(metaData, backupMeta)
	if err != nil {
		return commitTs, err
	}
	return backupMeta.EndVersion, nil
}

// constructOptions constructs options for BR and also return the remote path
func constructOptions(backup *v1alpha1.Backup) ([]string, string, error) {
	args, remotePath, err := util.ConstructBRGlobalOptionsForBackup(backup)
	if err != nil {
		return args, remotePath, err
	}
	config := backup.Spec.BR
	if config.Concurrency != nil {
		args = append(args, fmt.Sprintf("--concurrency=%d", *config.Concurrency))
	}
	if config.RateLimit != nil {
		args = append(args, fmt.Sprintf("--ratelimit=%d", *config.RateLimit))
	}
	if config.TimeAgo != "" {
		args = append(args, fmt.Sprintf("--timeago=%s", config.TimeAgo))
	}
	if config.Checksum != nil {
		args = append(args, fmt.Sprintf("--checksum=%t", *config.Checksum))
	}
	return args, remotePath, nil
}

// getBackupSize get the backup data size from remote
func getBackupSize(backup *v1alpha1.Backup) (int64, error) {
	var size int64
	s, err := util.NewRemoteStorage(backup)
	if err != nil {
		return size, err
	}
	defer s.Close()
	ctx := context.Background()
	iter := s.List(nil)
	for {
		obj, err := iter.Next(ctx)
		if err == io.EOF {
			break
		}
		if err != nil {
			return size, err
		}
		size += obj.Size
	}
	return size, nil
}
