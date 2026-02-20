# -*- coding: utf-8 -*-
"""
#  Araneae_WorkNode - test_views.py
"""
#  Copyright (c)   2024.12  Henry Zhao. All rights reserved.
#  From BJ.

# araneae_main - test_views.py
# Created by zhr62 at 2024/12/23 - 下午9:21
from unittest import TestCase
from Araneae_repo.views import check_project_code_files


class TestProjectViewSet(TestCase):
    def test_check_code_files(self):
        self.assertFalse(check_project_code_files(-1))
