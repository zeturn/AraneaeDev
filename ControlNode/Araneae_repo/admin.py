from django.contrib import admin

# Register your models here.

from .models import Project,Version,NodeProjectVersion
# Register your models here.

admin.site.register(Project)
admin.site.register(NodeProjectVersion)
admin.site.register(Version)