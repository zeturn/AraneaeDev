"""
Araneae_main/admin.py
"""
from django.contrib import admin

# Register your models here.

from .models import Workplace, OAuthIdentity

admin.site.register(Workplace)
admin.site.register(OAuthIdentity)
