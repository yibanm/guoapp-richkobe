import os
from dataclasses import dataclass


@dataclass(frozen=True)
class BuildVariant:
    # 单风味构建：始终包含全部站源，站源开关由应用设置控制
    all_sources: bool = True

    @property
    def slug(self):
        return 'feiguojian'

    @property
    def arguments(self):
        return []

    @property
    def flutter_arguments(self):
        return []

    @property
    def linker_flags(self):
        flags = '-s -w'
        update_server = os.environ.get('UPDATE_SERVER_URL', '').strip()
        if update_server:
            flags += ' -X duanjuapp/native/core.nativeUpdateServer=' + update_server
        return flags


def add_variant_argument(parser):
    parser.add_argument('--all-sources', action='store_true',
                        help='兼容参数：单风味始终含全部站源，此参数无实际效果')
