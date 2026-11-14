/**
 * Three.js objects built from the scene.
 *
 * The panel geometry is a convex fan: every panel the engine produces is
 * the sheet clipped by half-planes, so it is convex and a fan from the
 * first vertex triangulates it exactly. The UVs are the panel's
 * coordinates on the printed sheet, which is why the texture maps
 * correctly no matter how the panel folds.
 */

import * as THREE from 'three';

import type { Panel } from '../../bindings';

/** Builds a flat mesh geometry from a convex polygon and its UVs. */
export function buildGeometry(panel: Panel): THREE.BufferGeometry {
  const positions: number[] = [];
  const uvs: number[] = [];
  const indices: number[] = [];

  panel.polygon.forEach((point, index) => {
    positions.push(point.x, point.y, 0);
    const uv = panel.uv[index] ?? { x: 0, y: 0 };
    uvs.push(uv.x, uv.y);
  });
  for (let i = 1; i < panel.polygon.length - 1; i += 1) {
    indices.push(0, i, i + 1);
  }

  const geometry = new THREE.BufferGeometry();
  geometry.setAttribute('position', new THREE.Float32BufferAttribute(positions, 3));
  geometry.setAttribute('uv', new THREE.Float32BufferAttribute(uvs, 2));
  geometry.setIndex(indices);
  geometry.computeVertexNormals();
  return geometry;
}

/** The two textures taken from the printed pages. */
export interface Textures {
  outside: THREE.CanvasTexture;
  inside: THREE.CanvasTexture;
}

/** Creates the texture pair from the two printed pages. */
export function buildTextures(
  canvasA: HTMLCanvasElement,
  canvasB: HTMLCanvasElement,
): Textures {
  const outside = new THREE.CanvasTexture(canvasB);
  outside.colorSpace = THREE.SRGBColorSpace;
  outside.anisotropy = 4;

  // Side A is the mirror of the sheet frame, so its texture is flipped
  // horizontally before it is mapped with the sheet's own coordinates.
  const inside = new THREE.CanvasTexture(canvasA);
  inside.colorSpace = THREE.SRGBColorSpace;
  inside.anisotropy = 4;
  inside.wrapS = THREE.RepeatWrapping;
  inside.repeat.x = -1;
  inside.offset.x = 1;

  return { outside, inside };
}

/** A pair of material lists: one per mesh side. */
export interface Materials {
  outside: THREE.MeshLambertMaterial[];
  inside: THREE.MeshLambertMaterial[];
}

/** Creates the materials for one panel. */
export function buildMaterials(panel: Panel): {
  outside: THREE.MeshLambertMaterial;
  inside: THREE.MeshLambertMaterial;
} {
  const isSeam = panel.tint === 'seam';
  const offset = panel.layer * 1.5;

  // Coplanar layers would z-fight at t = 1. A polygon offset keeps the
  // order stable without moving the geometry, so the packet stays flat.
  const outside = new THREE.MeshLambertMaterial({
    side: THREE.FrontSide,
    color: isSeam ? 0xe9deca : 0xffffff,
    polygonOffset: true,
    polygonOffsetFactor: -offset,
    polygonOffsetUnits: -offset,
  });
  const inside = new THREE.MeshLambertMaterial({
    side: THREE.BackSide,
    color: isSeam ? 0xcfc4ae : 0xffffff,
    polygonOffset: true,
    polygonOffsetFactor: offset,
    polygonOffsetUnits: offset,
  });

  return { outside, inside };
}

/** Assigns the textures to the materials. */
export function applyTextures(materials: Materials, textures: Textures): void {
  materials.outside.forEach((material) => {
    material.map = textures.outside;
    material.needsUpdate = true;
  });
  materials.inside.forEach((material) => {
    material.map = textures.inside;
    material.needsUpdate = true;
  });
}

/** The viewer's mutable three.js state. */
export interface FoldStage {
  renderer: THREE.WebGLRenderer;
  scene: THREE.Scene;
  camera: THREE.PerspectiveCamera;
  folder: THREE.Group;
  groups: Map<string, THREE.Group>;
  materials: Materials;
}

/** Adds one panel's two meshes to a group. */
function addMeshes(stage: FoldStage, group: THREE.Group, panel: Panel): void {
  const geometry = buildGeometry(panel);
  const materials = buildMaterials(panel);
  stage.materials.outside.push(materials.outside);
  stage.materials.inside.push(materials.inside);

  const outside = new THREE.Mesh(geometry, materials.outside);
  const inside = new THREE.Mesh(geometry, materials.inside);
  outside.renderOrder = panel.layer;
  inside.renderOrder = panel.layer;
  group.add(outside, inside);
}

/** Disposes the panel tree and its materials. */
export function disposePanelTree(stage: FoldStage): void {
  stage.groups.forEach((group) => {
    group.traverse((object) => {
      if (object instanceof THREE.Mesh) {
        object.geometry.dispose();
      }
    });
    group.removeFromParent();
  });
  stage.groups.clear();
  stage.materials.outside.forEach((material) => material.dispose());
  stage.materials.inside.forEach((material) => material.dispose());
  stage.materials = { outside: [], inside: [] };
}

/**
 * Builds the hinge tree: the front panel's group is the root, every
 * other panel hangs off its parent's group.
 *
 * The groups are created parent-first, so a child always has a parent
 * group to attach to. The panel's geometry stays in absolute flat
 * coordinates; only the group matrices change while folding.
 */
export function buildPanelTree(stage: FoldStage, panels: Panel[]): void {
  disposePanelTree(stage);

  const root = panels.find((panel) => panel.parent === '');
  if (!root) {
    return;
  }

  const rootGroup = new THREE.Group();
  rootGroup.matrixAutoUpdate = false;
  stage.folder.add(rootGroup);
  stage.groups.set(root.id, rootGroup);
  addMeshes(stage, rootGroup, root);

  const pending = panels.filter((panel) => panel.parent !== '');
  let guard = 0;
  while (pending.length > 0 && guard < panels.length * 4) {
    guard += 1;
    for (let i = pending.length - 1; i >= 0; i -= 1) {
      const panel = pending[i];
      const parent = stage.groups.get(panel.parent);
      if (!parent) {
        continue;
      }

      const group = new THREE.Group();
      group.matrixAutoUpdate = false;
      parent.add(group);
      stage.groups.set(panel.id, group);
      addMeshes(stage, group, panel);
      pending.splice(i, 1);
    }
  }
}
