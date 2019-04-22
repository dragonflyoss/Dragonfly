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
	"path"

	cutil "github.com/dragonflyoss/Dragonfly/common/util"
	"github.com/dragonflyoss/Dragonfly/supernode/config"
)

func getDownloadPath(taskID string) string {
	return path.Join(config.DownloadHome, cutil.SubString(taskID, 0, 3), taskID)
}

func getMetaDataPath(taskID string) string {
	return path.Join(config.DownloadHome, cutil.SubString(taskID, 0, 3), taskID+".meta")
}

func getMd5DataPath(taskID string) string {
	return path.Join(config.DownloadHome, cutil.SubString(taskID, 0, 3), taskID+".md5")
}

func getUploadPath(taskID string) string {
	return path.Join(config.UploadHome, cutil.SubString(taskID, 0, 3), taskID)
}

func getHTTPPathStr(taskID string) string {
	return path.Join(config.HTTPSubPath, cutil.SubString(taskID, 0, 3), taskID)
}

func deleteTaskFiles(taskID string, deleteUploadFile bool) error {
	if err := cutil.DeleteFile(getDownloadPath(taskID)); err != nil {
		return err
	}

	if err := cutil.DeleteFile(getMetaDataPath(taskID)); err != nil {
		return err
	}

	if err := cutil.DeleteFile(getMd5DataPath(taskID)); err != nil {
		return err
	}

	if !deleteUploadFile {
		return nil
	}
	return cutil.DeleteFile(getUploadPath(taskID))
}
