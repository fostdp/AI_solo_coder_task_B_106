(function(window) {
    'use strict';

    const StoneModel = {
        createBuddhaStatue(seed = 1) {
            const group = new THREE.Group();
            const rng = this.seededRandom(seed);

            const stoneColor = new THREE.Color().setHSL(0.08 + rng() * 0.05, 0.3, 0.45);
            const stoneMat = new THREE.MeshStandardMaterial({
                color: stoneColor,
                roughness: 0.85,
                metalness: 0.05,
                flatShading: false
            });

            this._buildBase(group, stoneMat);
            this._buildTorso(group, stoneMat);
            this._buildHead(group, stoneMat, rng);
            this._buildLimbs(group, stoneMat);
            this._buildHalo(group, stoneMat);
            this.addWeathering(group, stoneMat, rng);

            return group;
        },

        _buildBase(group, stoneMat) {
            const baseGeo = new THREE.CylinderGeometry(4, 4.5, 1.5, 16);
            const base = new THREE.Mesh(baseGeo, stoneMat);
            base.position.y = 0.75;
            base.castShadow = true;
            base.receiveShadow = true;
            group.add(base);

            const pedestalGeo = new THREE.CylinderGeometry(3, 3.8, 2, 16);
            const pedestal = new THREE.Mesh(pedestalGeo, stoneMat);
            pedestal.position.y = 2.5;
            pedestal.castShadow = true;
            pedestal.receiveShadow = true;
            group.add(pedestal);

            const seatGeo = new THREE.CylinderGeometry(3.2, 3.2, 0.8, 16);
            const seat = new THREE.Mesh(seatGeo, stoneMat);
            seat.position.y = 3.9;
            seat.castShadow = true;
            group.add(seat);
        },

        _buildTorso(group, stoneMat) {
            const bodyGeo = new THREE.SphereGeometry(2.8, 32, 24, 0, Math.PI * 2, 0, Math.PI / 2.2);
            const body = new THREE.Mesh(bodyGeo, stoneMat);
            body.position.y = 6.5;
            body.scale.set(1, 1.3, 0.85);
            body.castShadow = true;
            group.add(body);

            const chestGeo = new THREE.BoxGeometry(3.5, 2.5, 1.8);
            const chest = new THREE.Mesh(chestGeo, stoneMat);
            chest.position.y = 6.2;
            chest.castShadow = true;
            group.add(chest);

            const robeGeo = new THREE.CylinderGeometry(2.8, 3.5, 3.5, 20);
            const robe = new THREE.Mesh(robeGeo, stoneMat);
            robe.position.y = 5.2;
            robe.castShadow = true;
            group.add(robe);

            const neckGeo = new THREE.CylinderGeometry(1, 1.1, 0.8, 16);
            const neck = new THREE.Mesh(neckGeo, stoneMat);
            neck.position.y = 8.2;
            neck.castShadow = true;
            group.add(neck);
        },

        _buildHead(group, stoneMat, rng) {
            const headGeo = new THREE.SphereGeometry(1.6, 32, 32);
            const head = new THREE.Mesh(headGeo, stoneMat);
            head.position.y = 10.2;
            head.scale.set(1, 1.1, 1);
            head.castShadow = true;
            group.add(head);

            const ushnishaGeo = new THREE.SphereGeometry(0.4, 16, 16);
            const ushnisha = new THREE.Mesh(ushnishaGeo, stoneMat);
            ushnisha.position.y = 11.8;
            ushnisha.scale.set(1, 1.5, 1);
            ushnisha.castShadow = true;
            group.add(ushnisha);

            for (let i = 0; i < 32; i++) {
                const curlGeo = new THREE.SphereGeometry(0.08, 6, 6);
                const curl = new THREE.Mesh(curlGeo, stoneMat);
                const phi = Math.acos(1 - i / 32 * 1.4);
                const theta = i * 1.8;
                const r = 1.65;
                curl.position.set(
                    head.position.x + r * Math.sin(phi) * Math.cos(theta),
                    head.position.y + r * Math.cos(phi) - 0.3,
                    head.position.z + r * Math.sin(phi) * Math.sin(theta)
                );
                group.add(curl);
            }

            const earGeo = new THREE.BoxGeometry(0.3, 1.2, 0.4);
            const leftEar = new THREE.Mesh(earGeo, stoneMat);
            leftEar.position.set(-1.8, 9.9, 0);
            leftEar.castShadow = true;
            group.add(leftEar);
            const rightEar = leftEar.clone();
            rightEar.position.x = 1.8;
            group.add(rightEar);
        },

        _buildLimbs(group, stoneMat) {
            const shoulderGeo = new THREE.SphereGeometry(1.2, 16, 16);
            const leftShoulder = new THREE.Mesh(shoulderGeo, stoneMat);
            leftShoulder.position.set(-2.5, 6.8, 0);
            leftShoulder.scale.set(1, 0.8, 1);
            leftShoulder.castShadow = true;
            group.add(leftShoulder);
            const rightShoulder = leftShoulder.clone();
            rightShoulder.position.x = 2.5;
            group.add(rightShoulder);

            const armGeo = new THREE.CylinderGeometry(0.5, 0.6, 3, 12);
            const leftArm = new THREE.Mesh(armGeo, stoneMat);
            leftArm.position.set(-3.2, 5, 1.5);
            leftArm.rotation.x = Math.PI / 3;
            leftArm.rotation.z = Math.PI / 8;
            leftArm.castShadow = true;
            group.add(leftArm);

            const rightArm = new THREE.Mesh(armGeo, stoneMat);
            rightArm.position.set(3.2, 5, 1.5);
            rightArm.rotation.x = Math.PI / 3;
            rightArm.rotation.z = -Math.PI / 8;
            rightArm.castShadow = true;
            group.add(rightArm);

            const handGeo = new THREE.SphereGeometry(0.6, 16, 16);
            const leftHand = new THREE.Mesh(handGeo, stoneMat);
            leftHand.position.set(-3.8, 3.5, 2.8);
            leftHand.scale.set(1, 0.7, 1.3);
            group.add(leftHand);
            const rightHand = leftHand.clone();
            rightHand.position.x = 3.8;
            group.add(rightHand);
        },

        _buildHalo(group, stoneMat) {
            const auraGeo = new THREE.RingGeometry(3.2, 3.5, 48);
            const auraMat = new THREE.MeshBasicMaterial({
                color: 0xd4a017,
                side: THREE.DoubleSide,
                transparent: true,
                opacity: 0.3
            });
            const aura = new THREE.Mesh(auraGeo, auraMat);
            aura.position.set(0, 9.5, -1);
            aura.rotation.x = 0.3;
            group.add(aura);
        },

        addSensorMarkers(sensors, latestData, camera) {
            const dataMap = {};
            latestData.forEach(d => { dataMap[d.sensor_id] = d; });
            const markers = [];

            sensors.forEach(s => {
                const isUS = s.type === 'ultrasonic';
                const data = dataMap[s.id];
                const value = data ? data.latest_value : 0;

                const scaleMax = isUS ? 4 : 60;
                const ratio = Math.min(value / scaleMax, 1);

                let color;
                if (isUS) {
                    if (value > 3) color = 0xf44336;
                    else if (value > 2) color = 0xff9800;
                    else if (value > 1) color = 0xffeb3b;
                    else color = 0x4caf50;
                } else {
                    if (value > 50) color = 0xf44336;
                    else if (value > 40) color = 0xff9800;
                    else color = 0x2196f3;
                }

                const geo = new THREE.SphereGeometry(isUS ? 0.15 : 0.12, 16, 16);
                const mat = new THREE.MeshStandardMaterial({
                    color: color,
                    emissive: color,
                    emissiveIntensity: 0.5 + ratio * 0.5,
                    roughness: 0.3,
                    metalness: 0.8
                });
                const marker = new THREE.Mesh(geo, mat);

                const angle = s.position_x * Math.PI * 2;
                const yPos = 2 + s.position_y * 10;
                const radius = 2.5 + Math.sin(yPos * 0.3) * 0.8;

                marker.position.set(
                    Math.cos(angle) * radius,
                    yPos,
                    Math.sin(angle) * radius
                );
                marker.userData = {
                    sensor: s,
                    data: data,
                    isUltrasonic: isUS,
                    originalY: yPos
                };
                markers.push(marker);

                const ringGeo = new THREE.RingGeometry(isUS ? 0.18 : 0.14, isUS ? 0.22 : 0.18, 32);
                const ringMat = new THREE.MeshBasicMaterial({
                    color: color,
                    side: THREE.DoubleSide,
                    transparent: true,
                    opacity: 0.6
                });
                const ring = new THREE.Mesh(ringGeo, ringMat);
                ring.position.copy(marker.position);
                ring.lookAt(camera.position);
                ring.userData = { isRing: true };
                markers.push(ring);
            });

            return markers;
        },

        addScaleOverlay(latestData) {
            const usData = latestData.filter(d => d.latest_unit === 'mm');
            if (usData.length === 0) {
                return null;
            }

            const values = usData.map(d => d.latest_value);
            const maxVal = Math.max(...values);
            const minVal = Math.min(...values);
            const range = maxVal - minVal || 1;

            const group = new THREE.Group();

            for (let lat = 0; lat < 20; lat++) {
                const phi = (lat / 20) * Math.PI;
                for (let lon = 0; lon < 32; lon++) {
                    const theta = (lon / 32) * Math.PI * 2;
                    const radius = 3;
                    const y = radius * Math.cos(phi);
                    const r = radius * Math.sin(phi);
                    const x = r * Math.cos(theta);
                    const z = r * Math.sin(theta);

                    const virtualPos = (y + 3) / 6;
                    const interpolated = this.interpolateValue(usData, virtualPos, lon / 32);
                    const normalizedVal = Math.min((interpolated - minVal) / range, 1);

                    const h = (1 - normalizedVal) * 0.3;
                    const s = 0.9;
                    const l = 0.35 + normalizedVal * 0.25;
                    const color = new THREE.Color().setHSL(h, s, l);

                    const dotGeo = new THREE.SphereGeometry(0.08, 6, 6);
                    const dotMat = new THREE.MeshBasicMaterial({
                        color: color,
                        transparent: true,
                        opacity: 0.75
                    });
                    const dot = new THREE.Mesh(dotGeo, dotMat);

                    const offset = 1 + normalizedVal * 0.3;
                    dot.position.set(x * offset, y * 0.8 + 5, z * offset);
                    group.add(dot);
                }
            }

            return group;
        },

        addWeathering(group, mat, rng) {
            const crackMat = new THREE.MeshBasicMaterial({ color: 0x2a1810 });
            for (let i = 0; i < 15; i++) {
                const crackGeo = new THREE.PlaneGeometry(0.02 + rng() * 0.05, 0.5 + rng() * 2);
                const crack = new THREE.Mesh(crackGeo, crackMat);
                crack.position.set(
                    (rng() - 0.5) * 6,
                    1 + rng() * 10,
                    1.8 + rng() * 1
                );
                crack.rotation.y = (rng() - 0.5) * 0.5;
                crack.rotation.z = (rng() - 0.5) * 0.5;
                group.add(crack);
            }

            for (let i = 0; i < 8; i++) {
                const chipGeo = new THREE.SphereGeometry(0.2 + rng() * 0.4, 8, 8);
                const chipMat = mat.clone();
                chipMat.color.setHex(0x1a1410);
                const chip = new THREE.Mesh(chipGeo, chipMat);
                chip.position.set(
                    (rng() - 0.5) * 5.5,
                    1 + rng() * 9,
                    (rng() - 0.5) * 3
                );
                chip.scale.set(
                    0.5 + rng() * 0.5,
                    0.3 + rng() * 0.4,
                    0.2 + rng() * 0.3
                );
                group.add(chip);
            }
        },

        seededRandom(seed) {
            let s = seed;
            return function() {
                s = (s * 9301 + 49297) % 233280;
                return s / 233280;
            };
        },

        interpolateValue(data, yPos, angle) {
            let totalDist = 0;
            let weightedSum = 0;
            data.forEach((d, i) => {
                const dataAngle = (i / Math.max(data.length - 1, 1)) ;
                const dataY = 0.3 + (i % 3) * 0.2;
                const dy = (yPos - dataY);
                const da = Math.abs(angle - dataAngle);
                const dist = Math.sqrt(dy*dy + Math.min(da, 1-da)*Math.min(da, 1-da));
                const weight = 1 / (dist * 5 + 0.1);
                weightedSum += d.latest_value * weight;
                totalDist += weight;
            });
            return totalDist > 0 ? weightedSum / totalDist : 0.5;
        }
    };

    window.StoneModel = StoneModel;
})(window);
