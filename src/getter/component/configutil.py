# coding=utf8
# Copyright 1999-2017 Alibaba Group.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
import os

client_config = {}


def _parser():
    """
    parse dragonfly config.
    [node]
    address=10.24.36.04,105.6.77.89

    """
    global client_config
    if os.path.exists("/etc/dragonfly.conf"):
        from ConfigParser import ConfigParser
        config = ConfigParser()
        config.read("/etc/dragonfly.conf")
        sections = config.sections()
        for section in sections:
            client_config[section] = dict(config.items(section))


_parser()
