/*
 * Copyright 1999-2017 Alibaba Group.
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
package com.alibaba.dragonfly.supernode.service.lock;

public interface LockConstants {

    String CDN_TRIGGER_LOCK = "cdn_trigger_lock_";
    String FILE_META_DATA_LOCK = "file_meta_data_lock_";
    String FILE_MD5_DATA_LOCK = "file_md5_data_lock_";
    String TASK_EXPIRE_LOCK = "task_expire_lock_";
}
