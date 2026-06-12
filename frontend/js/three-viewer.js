const ThreeViewer = {
    scene: null,
    camera: null,
    renderer: null,
    controls: null,
    relicGroup: null,
    sensors: [],
    scaleOverlay: null,
    wireframeMode: false,
    showSensors: true,
    showScaleOverlay: true,
    currentRelic: null,
    animating: false,

    init() {
        const container = document.getElementById('model-viewer');
        const width = container.clientWidth;
        const height = container.clientHeight;

        this.scene = new THREE.Scene();
        this.scene.background = new THREE.Color(0x0d1525);
        this.scene.fog = new THREE.Fog(0x0d1525, 50, 200);

        this.camera = new THREE.PerspectiveCamera(45, width / height, 0.1, 1000);
        this.camera.position.set(15, 12, 20);

        this.renderer = new THREE.WebGLRenderer({ antialias: true, alpha: true });
        this.renderer.setSize(width, height);
        this.renderer.setPixelRatio(window.devicePixelRatio);
        this.renderer.shadowMap.enabled = true;
        this.renderer.shadowMap.type = THREE.PCFSoftShadowMap;
        container.appendChild(this.renderer.domElement);

        this.controls = new THREE.OrbitControls(this.camera, this.renderer.domElement);
        this.controls.enableDamping = true;
        this.controls.dampingFactor = 0.05;
        this.controls.maxPolarAngle = Math.PI / 2 + 0.1;
        this.controls.minDistance = 5;
        this.controls.maxDistance = 80;

        this.setupLights();
        this.setupGround();
        this.relicGroup = new THREE.Group();
        this.scene.add(this.relicGroup);

        window.addEventListener('resize', () => this.onResize());
        this.onResize();

        this.setupControls();

        this.animate();
    },

    setupLights() {
        const ambient = new THREE.AmbientLight(0x404060, 0.5);
        this.scene.add(ambient);

        const hemi = new THREE.HemisphereLight(0x87ceeb, 0x8b7355, 0.4);
        this.scene.add(hemi);

        const dirLight = new THREE.DirectionalLight(0xffeedd, 1);
        dirLight.position.set(20, 30, 15);
        dirLight.castShadow = true;
        dirLight.shadow.mapSize.width = 2048;
        dirLight.shadow.mapSize.height = 2048;
        dirLight.shadow.camera.near = 0.5;
        dirLight.shadow.camera.far = 100;
        dirLight.shadow.camera.left = -30;
        dirLight.shadow.camera.right = 30;
        dirLight.shadow.camera.top = 30;
        dirLight.shadow.camera.bottom = -30;
        this.scene.add(dirLight);

        const fillLight = new THREE.DirectionalLight(0x88ccff, 0.3);
        fillLight.position.set(-15, 10, -10);
        this.scene.add(fillLight);

        const rimLight = new THREE.PointLight(0xffaa55, 0.5, 50);
        rimLight.position.set(0, 15, -15);
        this.scene.add(rimLight);
    },

    setupGround() {
        const groundGeo = new THREE.CircleGeometry(40, 64);
        const groundMat = new THREE.MeshStandardMaterial({
            color: 0x2a3450,
            roughness: 0.9,
            metalness: 0.1
        });
        const ground = new THREE.Mesh(groundGeo, groundMat);
        ground.rotation.x = -Math.PI / 2;
        ground.position.y = -0.01;
        ground.receiveShadow = true;
        this.scene.add(ground);

        const gridHelper = new THREE.GridHelper(40, 40, 0x3a4460, 0x252d45);
        gridHelper.position.y = 0.01;
        this.scene.add(gridHelper);
    },

    setupControls() {
        document.getElementById('show-wireframe').addEventListener('change', (e) => {
            this.wireframeMode = e.target.checked;
            this.updateWireframeMode();
        });
        document.getElementById('show-sensors').addEventListener('change', (e) => {
            this.showSensors = e.target.checked;
            this.sensors.forEach(s => s.visible = this.showSensors);
        });
        document.getElementById('show-scale-overlay').addEventListener('change', (e) => {
            this.showScaleOverlay = e.target.checked;
            if (this.scaleOverlay) this.scaleOverlay.visible = this.showScaleOverlay;
        });
    },

    async loadRelic(relic) {
        this.currentRelic = relic;
        document.getElementById('selected-relic-name').textContent =
            `🏛️ ${relic.name} (${relic.location})`;

        while (this.relicGroup.children.length > 0) {
            this.relicGroup.remove(this.relicGroup.children[0]);
        }
        this.sensors = [];

        const statue = StoneModel.createBuddhaStatue(relic.id);
        this.relicGroup.add(statue);

        if (relic.sensors && relic.sensors.length > 0) {
            const markers = StoneModel.addSensorMarkers(
                relic.sensors,
                relic.latest_data || [],
                this.camera
            );
            markers.forEach(m => {
                m.visible = this.showSensors;
                this.relicGroup.add(m);
                this.sensors.push(m);
            });
        }

        const overlay = StoneModel.addScaleOverlay(relic.latest_data || []);
        if (overlay) {
            overlay.visible = this.showScaleOverlay;
            this.relicGroup.add(overlay);
            this.scaleOverlay = overlay;
        }

        this.camera.position.set(18, 14, 22);
        this.controls.target.set(0, 6, 0);
        this.controls.update();
    },

    updateWireframeMode() {
        this.relicGroup.traverse(obj => {
            if (obj.isMesh && obj.material && !this.sensors.includes(obj) && obj !== this.scaleOverlay) {
                if (Array.isArray(obj.material)) {
                    obj.material.forEach(m => m.wireframe = this.wireframeMode);
                } else {
                    obj.material.wireframe = this.wireframeMode;
                }
            }
        });
    },

    onResize() {
        const container = document.getElementById('model-viewer');
        const width = container.clientWidth;
        const height = container.clientHeight;
        this.camera.aspect = width / height;
        this.camera.updateProjectionMatrix();
        this.renderer.setSize(width, height);
    },

    animate() {
        requestAnimationFrame(() => this.animate());
        this.controls.update();

        this.sensors.forEach(s => {
            if (s.userData && s.userData.isRing) {
                s.lookAt(this.camera.position);
            }
        });

        const time = Date.now() * 0.001;
        this.sensors.forEach(s => {
            if (s.userData && s.userData.sensor) {
                s.position.y = s.userData.originalY + Math.sin(time * 2 + s.userData.sensor.id) * 0.002;
            }
        });

        this.renderer.render(this.scene, this.camera);
    }
};
