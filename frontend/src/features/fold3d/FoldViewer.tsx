/**
 * The interactive 3D fold.
 *
 * Geometry is built once per scene: one convex polygon per panel, one
 * hinge group per panel, parented exactly as the engine's hinge tree.
 * Scrubbing only updates each group's local matrix and re-renders; no
 * geometry, material or texture is rebuilt per frame.
 *
 * The front panel is the root and its group never moves, which is what
 * makes "the front is flat and uncreased" structural rather than checked.
 */

import { useCallback, useEffect, useRef, useState } from 'react';
import * as THREE from 'three';
import { OrbitControls } from 'three/examples/jsm/controls/OrbitControls.js';

import type { Scene } from '../../bindings';
import { FoldControls, currentStepLabel } from './FoldControls';
import { worldMatrices, type FoldPanel } from './kinematics';
import {
  applyTextures,
  buildPanelTree,
  buildTextures,
  disposePanelTree,
  type FoldStage,
  type Textures,
} from './mesh';

/** Props of the viewer. */
export interface FoldViewerProps {
  scene: Scene;
  /** The printed Side A page, used as the inside texture. */
  canvasA: HTMLCanvasElement;
  /** The printed Side B page, used as the outside texture. */
  canvasB: HTMLCanvasElement;
  /** Increments whenever the canvases were redrawn. */
  textureVersion: number;
}

/** The play duration in milliseconds for a full fold. */
const PLAY_DURATION_MS = 3200;

/** The viewer component. */
export function FoldViewer({
  scene,
  canvasA,
  canvasB,
  textureVersion,
}: FoldViewerProps) {
  const mountRef = useRef<HTMLDivElement>(null);
  const stageRef = useRef<FoldStage | null>(null);
  const texturesRef = useRef<Textures | null>(null);
  const dirtyRef = useRef(true);

  const [progress, setProgress] = useState(0.35);
  const [playing, setPlaying] = useState(false);
  const directionRef = useRef(1);
  const lastFrameRef = useRef<number | null>(null);

  const foldPanelsRef = useRef<FoldPanel[]>([]);
  foldPanelsRef.current = scene.panels.map((panel) => ({
    id: panel.id,
    parent: panel.parent,
    hinge: panel.hinge,
    t_start: panel.t_start,
    t_end: panel.t_end,
  }));

  // --- Renderer, camera and lights -----------------------------------
  useEffect(() => {
    const mount = mountRef.current;
    if (!mount) {
      return;
    }

    const renderer = new THREE.WebGLRenderer({ antialias: true, alpha: true });
    renderer.setPixelRatio(Math.min(window.devicePixelRatio, 2));
    renderer.setSize(mount.clientWidth, Math.max(1, mount.clientHeight));
    mount.appendChild(renderer.domElement);

    const scene3 = new THREE.Scene();
    const camera = new THREE.PerspectiveCamera(
      40,
      mount.clientWidth / Math.max(1, mount.clientHeight),
      1,
      4000,
    );
    camera.position.set(40, 55, 250);

    const controls = new OrbitControls(camera, renderer.domElement);
    controls.enableDamping = true;
    controls.dampingFactor = 0.08;
    controls.minDistance = 60;
    controls.maxDistance = 900;
    controls.addEventListener('change', () => {
      dirtyRef.current = true;
    });

    scene3.add(new THREE.HemisphereLight(0xfff4dd, 0x2a251f, 2.1));
    const key = new THREE.DirectionalLight(0xffffff, 2.2);
    key.position.set(120, 180, 260);
    scene3.add(key);
    const rim = new THREE.DirectionalLight(0xffd9a0, 0.7);
    rim.position.set(-160, -120, 90);
    scene3.add(rim);

    const folder = new THREE.Group();
    folder.rotation.x = -0.32;
    folder.rotation.z = 0.05;
    scene3.add(folder);

    const stage: FoldStage = {
      renderer,
      scene: scene3,
      camera,
      folder,
      groups: new Map(),
      materials: { outside: [], inside: [] },
    };
    stageRef.current = stage;

    let frame = 0;
    const loop = () => {
      frame = requestAnimationFrame(loop);
      controls.update();
      if (dirtyRef.current) {
        dirtyRef.current = false;
        renderer.render(scene3, camera);
      }
    };
    loop();

    const resize = () => {
      const width = mount.clientWidth;
      const height = Math.max(1, mount.clientHeight);
      renderer.setSize(width, height);
      camera.aspect = width / height;
      camera.updateProjectionMatrix();
      dirtyRef.current = true;
    };
    const observer = new ResizeObserver(resize);
    observer.observe(mount);

    return () => {
      cancelAnimationFrame(frame);
      observer.disconnect();
      controls.dispose();
      disposePanelTree(stage);
      texturesRef.current?.outside.dispose();
      texturesRef.current?.inside.dispose();
      texturesRef.current = null;
      renderer.dispose();
      renderer.domElement.remove();
      stageRef.current = null;
    };
  }, []);

  // --- Geometry -------------------------------------------------------
  useEffect(() => {
    const stage = stageRef.current;
    if (!stage) {
      return;
    }
    buildPanelTree(stage, scene.panels);
    if (texturesRef.current) {
      applyTextures(stage.materials, texturesRef.current);
    }
    dirtyRef.current = true;
  }, [scene]);

  // --- Textures -------------------------------------------------------
  useEffect(() => {
    const stage = stageRef.current;
    if (!stage) {
      return;
    }
    texturesRef.current?.outside.dispose();
    texturesRef.current?.inside.dispose();
    texturesRef.current = buildTextures(canvasA, canvasB);
    applyTextures(stage.materials, texturesRef.current);
    dirtyRef.current = true;
  }, [canvasA, canvasB, textureVersion]);

  // --- Folding --------------------------------------------------------
  const applyProgress = useCallback((t: number) => {
    const stage = stageRef.current;
    if (!stage) {
      return;
    }
    const matrices = worldMatrices(foldPanelsRef.current, t);
    matrices.forEach((matrix, id) => {
      const group = stage.groups.get(id);
      if (group) {
        // The matrices are row-major, like the Go engine's; Matrix4.set
        // takes its arguments in row-major order, whereas fromArray
        // expects column-major.
        group.matrix.set(
          matrix[0], matrix[1], matrix[2], matrix[3],
          matrix[4], matrix[5], matrix[6], matrix[7],
          matrix[8], matrix[9], matrix[10], matrix[11],
          matrix[12], matrix[13], matrix[14], matrix[15],
        );
        group.matrixWorldNeedsUpdate = true;
      }
    });
    dirtyRef.current = true;
  }, []);

  useEffect(() => {
    applyProgress(progress);
  }, [applyProgress, progress, scene, textureVersion]);

  // Playback ping-pongs between folded and unfolded, which is what the
  // slider does by hand.
  useEffect(() => {
    if (!playing) {
      lastFrameRef.current = null;
      return;
    }

    let frame = 0;
    const step = (time: number) => {
      const previous = lastFrameRef.current ?? time;
      lastFrameRef.current = time;
      const delta = (time - previous) / PLAY_DURATION_MS;

      setProgress((current) => {
        let next = current + delta * directionRef.current;
        if (next >= 1) {
          next = 1;
          directionRef.current = -1;
        } else if (next <= 0) {
          next = 0;
          directionRef.current = 1;
        }
        return next;
      });

      frame = requestAnimationFrame(step);
    };
    frame = requestAnimationFrame(step);
    return () => cancelAnimationFrame(frame);
  }, [playing]);

  // A new scene starts from a partly folded pose, which shows both the
  // flat sheet and the packet in one glance.
  useEffect(() => {
    setProgress(0.35);
    setPlaying(false);
    directionRef.current = 1;
  }, [scene]);

  const metrics = scene.metrics;

  return (
    <div className="fold-viewer">
      <div className="fold-canvas" ref={mountRef} />

      <div className="fold-status">
        <span className="fold-step">{currentStepLabel(scene, progress)}</span>
        <span className="fold-metrics">
          {metrics.packet_width_mm.toFixed(1)} x {metrics.packet_height_mm.toFixed(1)} mm
          {' · '}
          {metrics.height_in.toFixed(2)} x {metrics.length_in.toFixed(2)} in
          {' · '}
          {metrics.within_limits
            ? 'inside the letter limits'
            : 'outside the letter limits'}
        </span>
      </div>

      <FoldControls
        scene={scene}
        progress={progress}
        playing={playing}
        onProgress={(value) => {
          setPlaying(false);
          setProgress(value);
        }}
        onTogglePlay={() => {
          directionRef.current = progress >= 1 ? -1 : 1;
          setPlaying((value) => !value);
        }}
      />

      <div className="fold-invariants">
        <span className="badge badge-good">Front: flat, one layer</span>
        <span className="badge badge-good">Creases: back only</span>
        <span className="badge badge-good">Address: computed diagonal</span>
        <span className="badge">{scene.panels.length} rigid panels</span>
      </div>
    </div>
  );
}

