# Hand-written migration: make gpu_info nullable on NodeCurrentStatus and NodeMetricArchive
# This matches models.py changes: gpu_info=JSONField(null=True, blank=True, default=list)

from django.db import migrations, models


class Migration(migrations.Migration):

    dependencies = [
        ('Araneae_manager', '0002_initial'),
    ]

    operations = [
        # NodeCurrentStatus.gpu_info: allow null + blank, default empty list
        migrations.AlterField(
            model_name='nodecurrentstatus',
            name='gpu_info',
            field=models.JSONField(blank=True, default=list, null=True),
        ),
        # NodeMetricArchive.gpu_info: allow null + blank, default empty list
        migrations.AlterField(
            model_name='nodemetricarchive',
            name='gpu_info',
            field=models.JSONField(blank=True, default=list, null=True),
        ),
    ]
