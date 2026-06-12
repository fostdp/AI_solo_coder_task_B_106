class CleaningRobot {
    constructor(scene) {
        this.scene = scene;
        this.group = new THREE.Group();
        this.laserBeam = null;
        this.laserSpot = null;
        this.pathPoints = [];
        this.currentPathIndex = 0;
        this.isMoving = false;
        this.laserActive = false;
        this.animationId = null;
        this.pathLine = null;
        this.cleanedAreas = [];

        this._buildBody();
        this.scene.add(this.group);
    }

    _buildBody() {
        const bodyMat = new THREE.MeshStandardMaterial({
            color: 0x2c5f8d,
            metalness: 0.7,
            roughness: 0.3
        });
        const accentMat = new THREE.MeshStandardMaterial({
            color: 0xff6b35,
            metalness: 0.8,
            roughness: 0.2
        });
        const darkMat = new THREE.MeshStandardMaterial({
            color: 0x1a1a1a,
            metalness: 0.5,
            roughness: 0.5
        });

        const baseGeom = new THREE.BoxGeometry(1.8, 0.5, 1.2);
        const base = new THREE.Mesh(baseGeom, bodyMat);
        base.position.y = 0.25;
        this.group.add(base);

        const wheelGeom = new THREE.CylinderGeometry(0.25, 0.25, 0.15, 24);
        const wheelPositions = [
            [-0.7, 0.25, 0.55], [0.7, 0.25, 0.55],
            [-0.7, 0.25, -0.55], [0.7, 0.25, -0.55]
        ];
        this.wheels = [];
        wheelPositions.forEach(pos => {
            const wheel = new THREE.Mesh(wheelGeom, darkMat);
            wheel.rotation.z = Math.PI / 2;
            wheel.position.set(pos[0], pos[1], pos[2]);
            this.group.add(wheel);
            this.wheels.push(wheel);
        });

        const armBaseGeom = new THREE.CylinderGeometry(0.25, 0.3, 0.4, 16);
        const armBase = new THREE.Mesh(armBaseGeom, accentMat);
        armBase.position.set(0, 0.7, 0);
        this.group.add(armBase);

        const arm1Geom = new THREE.BoxGeometry(0.2, 1.5, 0.2);
        this.arm1 = new THREE.Mesh(arm1Geom, bodyMat);
        this.arm1.position.set(0, 1.55, 0);
        this.arm1.rotation.z = -0.3;
        this.group.add(this.arm1);

        const jointGeom = new THREE.SphereGeometry(0.18, 16, 16);
        const joint1 = new THREE.Mesh(jointGeom, accentMat);
        joint1.position.set(0, 2.3, 0);
        this.arm1.add(joint1);

        const arm2Geom = new THREE.BoxGeometry(0.15, 1.2, 0.15);
        this.arm2 = new THREE.Mesh(arm2Geom, bodyMat);
        this.arm2.position.set(0.35, 2.9, 0);
        this.arm2.rotation.z = 0.8;
        this.group.add(this.arm2);

        const headGeom = new THREE.CylinderGeometry(0.12, 0.18, 0.4, 12);
        this.laserHead = new THREE.Mesh(headGeom, accentMat);
        this.laserHead.position.set(0, 0.6, 0);
        this.arm2.add(this.laserHead);

        const nozzleGeom = new THREE.ConeGeometry(0.08, 0.25, 12);
        const nozzle = new THREE.Mesh(nozzleGeom, darkMat);
        nozzle.position.set(0, 0.35, 0);
        nozzle.rotation.x = Math.PI;
        this.laserHead.add(nozzle);

        const panelGeom = new THREE.BoxGeometry(0.6, 0.4, 0.08);
        const panelMat = new THREE.MeshStandardMaterial({
            color: 0x00ff88,
            emissive: 0x00ff88,
            emissiveIntensity: 0.5
        });
        const panel = new THREE.Mesh(panelGeom, panelMat);
        panel.position.set(0, 0.85, -0.62);
        this.group.add(panel);

        this._setupLaser();
        this._setupLight();
    }

    _setupLaser() {
        const beamGeom = new THREE.CylinderGeometry(0.02, 0.04, 3, 8);
        const beamMat = new THREE.MeshBasicMaterial({
            color: 0xff2200,
            transparent: true,
            opacity: 0.0
        });
        this.laserBeam = new THREE.Mesh(beamGeom, beamMat);
        this.laserBeam.rotation.x = Math.PI / 2;
        this.laserBeam.position.set(0, 1.5, 0);
        this.laserHead.add(this.laserBeam);

        const spotGeom = new THREE.CircleGeometry(0.15, 24);
        const spotMat = new THREE.MeshBasicMaterial({
            color: 0xff4400,
            transparent: true,
            opacity: 0.0,
            side: THREE.DoubleSide
        });
        this.laserSpot = new THREE.Mesh(spotGeom, spotMat);
        this.laserSpot.rotation.x = -Math.PI / 2;
        this.laserSpot.position.set(0, 0.01, 0);
        this.scene.add(this.laserSpot);

        const glowGeom = new THREE.SphereGeometry(0.1, 16, 16);
        const glowMat = new THREE.MeshBasicMaterial({
            color: 0xff6600,
            transparent: true,
            opacity: 0.0
        });
        this.laserGlow = new THREE.Mesh(glowGeom, glowMat);
        this.laserHead.add(this.laserGlow);
        this.laserGlow.position.set(0, 0.4, 0);
    }

    _setupLight() {
        const robotLight = new THREE.PointLight(0x4488ff, 0.5, 8);
        robotLight.position.set(0, 2, 0);
        this.group.add(robotLight);
    }

    setPosition(x, y, z) {
        this.group.position.set(x, y, z);
    }

    setRotation(x, y, z) {
        this.group.rotation.set(x, y, z);
    }

    setLaserActive(active) {
        this.laserActive = active;
        const targetOpacity = active ? 0.85 : 0.0;
        const spotOpacity = active ? 0.9 : 0.0;

        if (this.laserBeam) this.laserBeam.material.opacity = targetOpacity;
        if (this.laserSpot) this.laserSpot.material.opacity = spotOpacity;
        if (this.laserGlow) this.laserGlow.material.opacity = active ? 0.6 : 0.0;

        if (active) {
            const worldPos = new THREE.Vector3();
            this.laserHead.getWorldPosition(worldPos);
            const dir = new THREE.Vector3(0, -1, 0);
            dir.applyQuaternion(this.laserHead.getWorldQuaternion(new THREE.Quaternion()));
            const raycaster = new THREE.Raycaster(worldPos, dir.normalize(), 0, 10);
            const hits = raycaster.intersectObjects(this.scene.children, true);
            if (hits.length > 0) {
                this.laserSpot.position.copy(hits[0].point);
                this.laserSpot.position.y += 0.01;
                this._addCleanedArea(hits[0].point);
            }
        }
    }

    _addCleanedArea(point) {
        const areaGeom = new THREE.CircleGeometry(0.25, 24);
        const areaMat = new THREE.MeshBasicMaterial({
            color: 0x88ff88,
            transparent: true,
            opacity: 0.4,
            side: THREE.DoubleSide
        });
        const area = new THREE.Mesh(areaGeom, areaMat);
        area.rotation.x = -Math.PI / 2;
        area.position.set(point.x, point.y + 0.015, point.z);
        this.scene.add(area);
        this.cleanedAreas.push(area);
    }

    clearCleanedAreas() {
        this.cleanedAreas.forEach(a => this.scene.remove(a));
        this.cleanedAreas = [];
    }

    showPath(points) {
        this.hidePath();
        this.pathPoints = points;

        const geom = new THREE.BufferGeometry();
        const positions = [];
        points.forEach(p => {
            positions.push(p.x, p.y + 0.1, p.z);
        });
        geom.setAttribute('position', new THREE.Float32BufferAttribute(positions, 3));

        const mat = new THREE.LineDashedMaterial({
            color: 0x00ffff,
            dashSize: 0.3,
            gapSize: 0.15,
            transparent: true,
            opacity: 0.7
        });

        this.pathLine = new THREE.Line(geom, mat);
        this.pathLine.computeLineDistances();
        this.scene.add(this.pathLine);

        points.forEach((p, i) => {
            const markerGeom = new THREE.SphereGeometry(0.08, 12, 12);
            const hue = i / Math.max(points.length - 1, 1);
            const markerMat = new THREE.MeshBasicMaterial({
                color: new THREE.Color().setHSL(hue, 1, 0.5)
            });
            const marker = new THREE.Mesh(markerGeom, markerMat);
            marker.position.set(p.x, p.y + 0.15, p.z);
            marker.userData.isPathMarker = true;
            this.scene.add(marker);

            const ringGeom = new THREE.RingGeometry(0.1, 0.14, 16);
            const ringMat = new THREE.MeshBasicMaterial({
                color: 0xffffff,
                side: THREE.DoubleSide,
                transparent: true,
                opacity: 0.6
            });
            const ring = new THREE.Mesh(ringGeom, ringMat);
            ring.rotation.x = -Math.PI / 2;
            ring.position.set(p.x, p.y + 0.12, p.z);
            ring.userData.isPathMarker = true;
            this.scene.add(ring);
        });
    }

    hidePath() {
        if (this.pathLine) {
            this.scene.remove(this.pathLine);
            this.pathLine = null;
        }
        const toRemove = [];
        this.scene.traverse(obj => {
            if (obj.userData && obj.userData.isPathMarker) {
                toRemove.push(obj);
            }
        });
        toRemove.forEach(o => this.scene.remove(o));
    }

    followPath(points, onComplete, onProgress) {
        if (!points || points.length === 0) return;
        this.pathPoints = points;
        this.currentPathIndex = 0;
        this.isMoving = true;
        this.showPath(points);
        this.clearCleanedAreas();

        const moveToNext = () => {
            if (this.currentPathIndex >= points.length) {
                this.isMoving = false;
                this.setLaserActive(false);
                if (onComplete) onComplete();
                return;
            }

            const target = points[this.currentPathIndex];
            const startPos = this.group.position.clone();
            const endPos = new THREE.Vector3(target.x, target.y, target.z);
            const duration = 1200;
            const startTime = performance.now();

            const dx = target.x - startPos.x;
            const dz = target.z - startPos.z;
            const targetRotY = Math.atan2(dx, dz);

            const animate = () => {
                const elapsed = performance.now() - startTime;
                const t = Math.min(elapsed / duration, 1);
                const easeT = t < 0.5 ? 2 * t * t : 1 - Math.pow(-2 * t + 2, 2) / 2;

                this.group.position.lerpVectors(startPos, endPos, easeT);
                this.group.rotation.y += (targetRotY - this.group.rotation.y) * 0.1;

                this.wheels.forEach(w => {
                    w.rotation.x += 0.05;
                });

                if (t > 0.85 && t < 0.98) {
                    this.setLaserActive(true);
                } else {
                    this.setLaserActive(false);
                }

                if (onProgress) {
                    onProgress((this.currentPathIndex + t) / points.length, this.currentPathIndex);
                }

                if (t < 1) {
                    this.animationId = requestAnimationFrame(animate);
                } else {
                    this.currentPathIndex++;
                    setTimeout(moveToNext, 200);
                }
            };

            animate();
        };

        moveToNext();
    }

    stop() {
        this.isMoving = false;
        if (this.animationId) {
            cancelAnimationFrame(this.animationId);
            this.animationId = null;
        }
        this.setLaserActive(false);
    }

    dispose() {
        this.stop();
        this.hidePath();
        this.clearCleanedAreas();
        this.scene.remove(this.group);
    }
}
