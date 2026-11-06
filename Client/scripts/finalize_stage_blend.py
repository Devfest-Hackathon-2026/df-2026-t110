"""Turn one library-written asset .blend into an openable Blender scene.

Run as: blender -b path/to/asset.blend --python finalize_stage_blend.py
"""

import bpy
import os

collection = bpy.data.collections.get("RF_EXPORT_TEMP")
if collection is None:
    raise RuntimeError("Missing exported stage collection")
scene = bpy.context.scene
if collection.name not in scene.collection.children:
    scene.collection.children.link(collection)
scene.name = "RF_" + os.path.splitext(os.path.basename(bpy.data.filepath))[0]
bpy.context.preferences.filepaths.save_version = 0
bpy.ops.wm.save_as_mainfile(filepath=bpy.data.filepath)
print("RF_SOURCE_READY", bpy.data.filepath, len(scene.objects))
