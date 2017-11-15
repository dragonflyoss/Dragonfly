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
class ParamError(Exception):
    """
    param is invalid
    """

    def __init__(self, msg=''):
        Exception.__init__(self, msg)


class DownError(Exception):
    """
    down error
    """

    def __init__(self, msg=''):
        Exception.__init__(self, msg)


class DirError(Exception):
    """
    dir create/delete and so on error
    """

    def __init__(self, msg=''):
        Exception.__init__(self, msg)


class SpaceLackError(Exception):
    """
    disk space is lack
    """

    def __init__(self, msg=''):
        Exception.__init__(self, msg)


class Md5NotMatchError(Exception):
    """
    md5 not match
    """

    def __init__(self, msg=''):
        Exception.__init__(self, msg)


class FileIOError(Exception):
    """
    file io error
    """

    def __init__(self, msg=''):
        Exception.__init__(self, msg)


class ReadTimeoutError(Exception):
    """
    read http response timeout
    """

    def __init__(self, msg=''):
        Exception.__init__(self, msg)


class NeedBack(Exception):
    """
    need back source to download
    """

    def __init__(self, msg=''):
        Exception.__init__(self, msg)
