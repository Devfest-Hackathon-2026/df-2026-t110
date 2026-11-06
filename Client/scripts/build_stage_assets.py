"""Build textured stage GLBs in the connected Blender scene through Blender MCP.

The currently open robot model and .blend file are left untouched. The written
asset .blend libraries must be finalized with finalize_stage_blend.py before
opening them as ordinary Blender scenes.
"""

import bpy
import math
import os
import random
from mathutils import Vector

OUTPUT = os.environ.get(
    "ROBOT_FIELD_STAGE_ASSETS",
    r"C:\Users\yuxtu\Programming-Learning-AR\Client\public\stage-assets",
)
SOURCE = os.environ.get(
    "ROBOT_FIELD_BLEND_ASSETS",
    os.path.join(os.path.dirname(os.path.dirname(OUTPUT)), "assets", "blender"),
)
TEXTURES = os.path.join(OUTPUT, "textures")
os.makedirs(TEXTURES, exist_ok=True)
os.makedirs(SOURCE, exist_ok=True)
selected_before = list(bpy.context.selected_objects)
active_before = bpy.context.view_layer.objects.active
temp = bpy.data.collections.new("RF_EXPORT_TEMP")
bpy.context.scene.collection.children.link(temp)
materials = []
images = []
generated = []


def texture(name, base, grain, style):
    image = bpy.data.images.new("RF_" + name, 128, 128, alpha=True)
    pixels = []
    for y in range(128):
        v = y / 127
        for x in range(128):
            u = x / 127
            noise = (math.sin(u * 41 + v * 19) * 0.36 + math.sin(u * 107 - v * 67) * 0.18 + math.sin(u * 13 + v * 35) * 0.46)
            detail = 0
            if style == "stone":
                detail = -0.13 if abs(math.sin(u * 18 + math.sin(v * 11) * 2.4)) < 0.035 else 0.025 * math.sin(v * 50 + u * 8)
            elif style == "brass":
                detail = 0.045 * math.sin(v * 145 + 4 * math.sin(u * 6))
            elif style == "parcel":
                detail = 0.055 if abs(math.sin((u + v * 0.57) * 22)) < 0.045 else 0
            elif style == "circuit":
                detail = 0.035 if min(abs((u * 8) % 1 - 0.5), abs((v * 8) % 1 - 0.5)) < 0.018 and (u < 0.18 or v > 0.78) else 0
            pixels.extend([max(0, min(1, c + grain * noise + detail)) for c in base] + [1])
    image.pixels.foreach_set(pixels)
    image.filepath_raw = os.path.join(TEXTURES, name + ".png")
    image.file_format = "PNG"
    image.save()
    image.pack()
    images.append(image)
    return image


def surface(name, image=None, color=(1, 1, 1, 1), metal=0, rough=0.5, glow=None):
    mat = bpy.data.materials.new("RF_" + name)
    mat.diffuse_color = color
    mat.use_nodes = True
    shader = mat.node_tree.nodes.get("Principled BSDF")
    shader.inputs["Base Color"].default_value = color
    shader.inputs["Metallic"].default_value = metal
    shader.inputs["Roughness"].default_value = rough
    if image:
        source = mat.node_tree.nodes.new("ShaderNodeTexImage")
        source.image = image
        mat.node_tree.links.new(source.outputs["Color"], shader.inputs["Base Color"])
    if glow:
        shader.inputs["Emission Color"].default_value = (*glow, 1)
        shader.inputs["Emission Strength"].default_value = 1.8
    materials.append(mat)
    return mat


tile = surface("ceramic-deck", texture("ceramic-deck", (0.29, 0.43, 0.40), 0.028, "circuit"), metal=0.36, rough=0.48)
rock = surface("fractured-rock", texture("fractured-rock", (0.40, 0.48, 0.51), 0.085, "stone"), metal=0.12, rough=0.74)
brass = surface("brushed-brass", texture("brushed-brass", (0.65, 0.43, 0.17), 0.024, "brass"), metal=0.83, rough=0.27)
parcel = surface("memory-parcel", texture("memory-parcel", (0.56, 0.82, 0.66), 0.025, "parcel"), metal=0.18, rough=0.35)
violet = surface("checkpoint-enamel", texture("checkpoint-enamel", (0.36, 0.36, 0.61), 0.029, "circuit"), metal=0.45, rough=0.32)
dark = surface("graphite-frame", color=(0.035, 0.09, 0.11, 1), metal=0.57, rough=0.38)
cut = surface("rock-cut-face", color=(0.22, 0.32, 0.37, 1), metal=0.23, rough=0.52)
mint = surface("mint-light", color=(0.12, 0.9, 0.64, 1), rough=0.2, glow=(0.15, 1, 0.65))
blue = surface("blue-light", color=(0.12, 0.64, 1, 1), rough=0.2, glow=(0.10, 0.55, 1))
gold = surface("gold-light", color=(1, 0.61, 0.13, 1), rough=0.22, glow=(1, 0.56, 0.09))


def collect(obj, mat, smooth=False):
    for collection in list(obj.users_collection):
        collection.objects.unlink(obj)
    temp.objects.link(obj)
    obj.data.materials.append(mat)
    if smooth:
        for face in obj.data.polygons:
            face.use_smooth = True
    return obj


def bevel(obj, amount=0.02):
    edge = obj.modifiers.new("machined edges", "BEVEL")
    edge.width = amount
    edge.segments = 2
    normal = obj.modifiers.new("weighted normals", "WEIGHTED_NORMAL")
    normal.keep_sharp = True
    return obj


def box(name, location, size, mat, edge=0):
    bpy.ops.mesh.primitive_cube_add(size=1, location=location)
    obj = collect(bpy.context.object, mat)
    obj.name = "RF_" + name
    obj.dimensions = size
    bpy.ops.object.transform_apply(location=False, rotation=False, scale=True)
    if edge:
        bevel(obj, edge)
    return obj


def cylinder(name, location, radius, depth, mat, sides=8, edge=0):
    bpy.ops.mesh.primitive_cylinder_add(vertices=sides, radius=radius, depth=depth, location=location)
    obj = collect(bpy.context.object, mat, True)
    obj.name = "RF_" + name
    if edge:
        bevel(obj, edge)
    return obj


def torus(name, location, major, minor, mat, vertical=False):
    bpy.ops.mesh.primitive_torus_add(major_segments=48, minor_segments=8, location=location, major_radius=major, minor_radius=minor, rotation=(math.pi / 2, 0, 0) if vertical else (0, 0, 0))
    obj = collect(bpy.context.object, mat, True)
    obj.name = "RF_" + name
    return obj


def rod(name, a, b, radius, mat):
    a, b = Vector(a), Vector(b)
    obj = cylinder(name, (a + b) / 2, radius, (b - a).length, mat, 7)
    obj.rotation_euler = (b - a).to_track_quat("Z", "Y").to_euler()
    return obj


def shard(name, location, radius, height, mat, sides=7, seed=0, top_scale=0.12):
    rng = random.Random(seed)
    vertices = []
    for z, scale in [(0, 0.75), (height * 0.48, 1), (height, top_scale)]:
        for i in range(sides):
            angle = i * math.tau / sides
            r = radius * scale * rng.uniform(0.82, 1.16)
            vertices.append((math.cos(angle) * r, math.sin(angle) * r, z + rng.uniform(-0.018, 0.018)))
    faces = [tuple(reversed(range(sides)))]
    for ring in range(2):
        for i in range(sides):
            a, b = ring * sides + i, ring * sides + (i + 1) % sides
            faces.append((a, b, b + sides, a + sides))
    faces.append(tuple(range(2 * sides, 3 * sides)))
    mesh = bpy.data.meshes.new("RF_" + name + "_mesh")
    mesh.from_pydata(vertices, [], faces)
    mesh.update()
    uv = mesh.uv_layers.new(name="stone mapping")
    for face in mesh.polygons:
        for loop_index in face.loop_indices:
            p = mesh.vertices[mesh.loops[loop_index].vertex_index].co
            uv.data[loop_index].uv = (p.x / (radius * 2) + 0.5, p.z / height)
    obj = bpy.data.objects.new("RF_" + name, mesh)
    temp.objects.link(obj)
    obj.location = location
    obj.data.materials.append(mat)
    return obj


def export(name):
    bpy.ops.object.select_all(action="DESELECT")
    objects = list(temp.objects)
    meshes = [obj.data for obj in objects if obj.type == "MESH"]
    for obj in objects:
        obj.select_set(True)
    bpy.context.view_layer.objects.active = objects[0]
    path = os.path.join(OUTPUT, name + ".glb")
    bpy.ops.export_scene.gltf(filepath=path, export_format="GLB", use_selection=True)
    source_scene = bpy.data.scenes.new("RF_" + name)
    source_scene.collection.children.link(temp)
    bpy.data.libraries.write(
        os.path.join(SOURCE, name + ".blend"), {source_scene},
        path_remap="RELATIVE_ALL", compress=True,
    )
    source_scene.collection.children.unlink(temp)
    bpy.data.scenes.remove(source_scene)
    generated.append({"name": name, "meshes": len(objects), "bytes": os.path.getsize(path)})
    for obj in objects:
        bpy.data.objects.remove(obj, do_unlink=True)
    for mesh in meshes:
        if mesh.users == 0:
            bpy.data.meshes.remove(mesh)


# Layered circuit tile, top at about z=0.05.
box("floor-shell", (0, 0, -0.11), (0.96, 0.96, 0.18), dark, 0.025)
box("floor-bezel", (0, 0, 0.002), (0.90, 0.90, 0.045), tile, 0.035)
box("floor-inset", (0, 0, 0.036), (0.72, 0.72, 0.018), tile, 0.018)
for side in [-1, 1]:
    box("floor-bus-a", (side * 0.38, 0, 0.048), (0.022, 0.56, 0.012), mint, 0.004)
    box("floor-bus-b", (0, side * 0.38, 0.048), (0.56, 0.022, 0.012), mint, 0.004)
for x in [-0.38, 0.38]:
    for y in [-0.38, 0.38]:
        cylinder("floor-fastener", (x, y, 0.05), 0.018, 0.011, brass, 12)
export("floor")

# Asymmetric fractured obstacle with embedded glowing fissures.
shard("rock-main", (-0.04, 0.02, 0.045), 0.34, 0.69, rock, 9, 71, 0.47)
shard("rock-side-a", (0.27, -0.13, 0.045), 0.19, 0.48, cut, 7, 52, 0.38)
shard("rock-side-b", (-0.28, 0.14, 0.045), 0.18, 0.43, rock, 6, 98, 0.4)
shard("rock-vein", (0.13, 0.18, 0.18), 0.07, 0.43, mint, 5, 22)
for i, (a, b) in enumerate([((-0.19, -0.21, 0.14), (-0.04, -0.25, 0.41)), ((-0.04, -0.25, 0.41), (0.13, -0.22, 0.51)), ((0.07, 0.30, 0.18), (0.22, 0.18, 0.39))]):
    rod("rock-fissure-" + str(i), a, b, 0.008, blue)
export("rock")

# Powered transfer arch: the endpoint is a three-dimensional device.
cylinder("goal-foot", (0, 0.12, 0.095), 0.32, 0.09, brass, 8, 0.018)
cylinder("goal-plinth", (0, 0.12, 0.15), 0.25, 0.045, dark, 8, 0.012)
torus("goal-portal", (0, 0.12, 0.43), 0.22, 0.045, brass, True)
torus("goal-inner-light", (0, 0.12, 0.43), 0.166, 0.009, gold, True)
torus("goal-halo", (0, 0.12, 0.43), 0.273, 0.012, mint, True)
for x in [-0.2, 0.2]:
    rod("goal-support", (x, 0.12, 0.17), (x * 0.76, 0.12, 0.29), 0.026, brass)
export("goal")

# Ordered checkpoint relay; its number is supplied dynamically by the client.
cylinder("checkpoint-foot", (0, 0, 0.09), 0.28, 0.085, violet, 8, 0.018)
cylinder("checkpoint-inner", (0, 0, 0.15), 0.19, 0.045, dark, 8, 0.012)
for x, y in [(0.15, 0), (-0.15, 0), (0, 0.15), (0, -0.15)]:
    rod("checkpoint-prong", (x, y, 0.17), (x * 0.65, y * 0.65, 0.47), 0.022, violet)
torus("checkpoint-orbit", (0, 0, 0.49), 0.18, 0.021, blue)
shard("checkpoint-core", (0, 0, 0.29), 0.085, 0.29, mint, 6, 34)
export("checkpoint")

# Transportable data packet with a tactile shell, straps and illuminated window.
box("parcel-shell", (0, 0, 0.27), (0.47, 0.39, 0.32), parcel, 0.052)
box("parcel-top", (0, 0, 0.443), (0.35, 0.28, 0.024), dark, 0.01)
box("parcel-window", (0, 0, 0.458), (0.22, 0.15, 0.015), mint, 0.008)
for x in [-0.18, 0.18]:
    box("parcel-strap", (x, 0, 0.28), (0.036, 0.42, 0.37), brass, 0.012)
for y in [-0.145, 0.145]:
    box("parcel-handle", (0, y, 0.50), (0.20, 0.025, 0.022), brass, 0.007)
export("luggage")

# Legacy drop station retained for the symbol-map sample lessons.
cylinder("dock-foot", (0, 0, 0.09), 0.31, 0.075, brass, 8, 0.016)
cylinder("dock-bed", (0, 0, 0.13), 0.25, 0.025, dark, 8, 0.008)
torus("dock-target", (0, 0, 0.16), 0.19, 0.018, gold)
for angle in [0, math.pi / 2, math.pi, 3 * math.pi / 2]:
    x, y = math.cos(angle) * 0.29, math.sin(angle) * 0.29
    box("dock-clamp", (x, y, 0.17), (0.085, 0.055, 0.07), brass, 0.01)
export("dock")

# Low start beacon beneath the robot.
cylinder("start-foot", (0, 0, 0.078), 0.31, 0.055, dark, 8, 0.01)
torus("start-ring", (0, 0, 0.116), 0.24, 0.017, mint)
for angle in [0, math.pi / 2, math.pi, 3 * math.pi / 2]:
    rod("start-spoke", (0, 0, 0.12), (math.cos(angle) * 0.21, math.sin(angle) * 0.21, 0.12), 0.009, blue)
export("start")

# Sea frame leaves the middle open for the client's animated water shader.
for side in [-1, 1]:
    box("sea-frame-x", (0, side * 0.46, -0.05), (0.96, 0.045, 0.24), dark, 0.014)
    box("sea-frame-y", (side * 0.46, 0, -0.05), (0.045, 0.86, 0.24), dark, 0.014)
    box("sea-light-x", (0, side * 0.42, 0.077), (0.72, 0.012, 0.009), blue, 0.003)
    box("sea-light-y", (side * 0.42, 0, 0.077), (0.012, 0.72, 0.009), blue, 0.003)
export("sea-frame")

bpy.data.collections.remove(temp)
for mat in materials:
    bpy.data.materials.remove(mat)
for image in images:
    bpy.data.images.remove(image)
bpy.ops.object.select_all(action="DESELECT")
for obj in selected_before:
    if obj.name in bpy.data.objects:
        obj.select_set(True)
if active_before and active_before.name in bpy.data.objects:
    bpy.context.view_layer.objects.active = active_before
result = {"generated": generated, "original_objects": [o.name for o in bpy.context.scene.objects]}
