/*
 * Copyright The Dragonfly Authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package cdn

import (
	"context"
	"path"

	cutil "github.com/dragonflyoss/Dragonfly/common/util"
	"github.com/dragonflyoss/Dragonfly/supernode/config"
	"github.com/dragonflyoss/Dragonfly/supernode/store"
)

var getDownloadRawFunc = getDownloadRaw
var getMetaDataRawFunc = getMetaDataRaw
var getMd5DataRawFunc = getMd5DataRaw

func getDownloadKey(taskID string) string {
	return path.Join(cutil.SubString(taskID, 0, 3), taskID)
}

func getMetaDataKey(taskID string) string {
	return path.Join(cutil.SubString(taskID, 0, 3), taskID+".meta")
}

func getMd5DataKey(taskID string) string {
	return path.Join(cutil.SubString(taskID, 0, 3), taskID+".md5")
}

func getUploadKey(taskID string) string {
	return path.Join(cutil.SubString(taskID, 0, 3), taskID)
}

func getDownloadRaw(taskID string) *store.Raw {
	return &store.Raw{
		Bucket: config.DownloadHome,
		Key:    getDownloadKey(taskID),
	}
}

func getMetaDataRaw(taskID string) *store.Raw {
	return &store.Raw{
		Bucket: config.DownloadHome,
		Key:    getMetaDataKey(taskID),
	}
}

func getMd5DataRaw(taskID string) *store.Raw {
	return &store.Raw{
		Bucket: config.DownloadHome,
		Key:    getMd5DataKey(taskID),
	}
}

func getUploadRaw(taskID string) *store.Raw {
	return &store.Raw{
		Bucket: config.DownloadHome,
		Key:    getUploadKey(taskID),
	}
}

func deleteTaskFiles(ctx context.Context, cacheStore *store.Store, taskID string, deleteUploadFile bool) error {
	if err := cacheStore.Remove(ctx, getMetaDataRaw(taskID)); err != nil &&
		!store.IsKeyNotFound(err) {
		return err
	}

	if err := cacheStore.Remove(ctx, getMd5DataRaw(taskID)); err != nil &&
		!store.IsKeyNotFound(err) {
		return err
	}

	if err := cacheStore.Remove(ctx, getDownloadRaw(taskID)); err != nil &&
		!store.IsKeyNotFound(err) {
		return err
	}

	if deleteUploadFile {
		if err := cacheStore.Remove(ctx, getUploadRaw(taskID)); err != nil &&
			!store.IsKeyNotFound(err) {
			return err
		}
	}
	return nil
}
