(function (global) {
    'use strict';

    function TextureLoader() {
        this._cache = new Map();
        this._ktx2Supported = this._detectKTX2Support();
        this._basisSupported = this._detectBasisSupport();
    }

    TextureLoader.prototype._detectKTX2Support = function () {
        try {
            var canvas = document.createElement('canvas');
            var gl = canvas.getContext('webgl2') || canvas.getContext('webgl');
            if (!gl) return false;

            var ext = gl.getExtension('WEBGL_compressed_texture_astc') ||
                      gl.getExtension('WEBGL_compressed_texture_etc') ||
                      gl.getExtension('WEBGL_compressed_texture_pvrtc') ||
                      gl.getExtension('WEBGL_compressed_texture_s3tc') ||
                      gl.getExtension('WEBGL_compressed_texture_s3tc_srgb');
            return !!ext;
        } catch (e) {
            return false;
        }
    };

    TextureLoader.prototype._detectBasisSupport = function () {
        return typeof THREE !== 'undefined' &&
               typeof THREE.KTX2Loader !== 'undefined';
    };

    TextureLoader.prototype._getPreferredFormat = function () {
        if (this._basisSupported) return '.ktx2';
        if (this._ktx2Supported) return '.ktx2';
        return '.png';
    };

    TextureLoader.prototype.load = function (basePath, options) {
        var self = this;
        options = options || {};
        var useCompression = options.compression !== false;
        var format = useCompression ? this._getPreferredFormat() : '.png';
        var fullPath = basePath + format;

        if (this._cache.has(fullPath)) {
            return Promise.resolve(this._cache.get(fullPath));
        }

        return new Promise(function (resolve, reject) {
            if (typeof THREE === 'undefined') {
                var img = new Image();
                img.crossOrigin = 'anonymous';
                img.onload = function () {
                    self._cache.set(fullPath, img);
                    resolve(img);
                };
                img.onerror = function () {
                    if (format !== '.png') {
                        var pngPath = basePath + '.png';
                        var fallback = new Image();
                        fallback.crossOrigin = 'anonymous';
                        fallback.onload = function () {
                            self._cache.set(fullPath, fallback);
                            resolve(fallback);
                        };
                        fallback.onerror = reject;
                        fallback.src = pngPath;
                    } else {
                        reject(new Error('Failed to load texture: ' + fullPath));
                    }
                };
                img.src = fullPath;
            } else {
                var loader;
                if (format === '.ktx2' && typeof THREE.KTX2Loader !== 'undefined') {
                    loader = new THREE.KTX2Loader();
                    if (options.transcoderPath) {
                        loader.setTranscoderPath(options.transcoderPath);
                    }
                    if (options.renderer) {
                        loader.detectSupport(options.renderer);
                    }
                } else {
                    loader = new THREE.TextureLoader();
                }

                loader.load(
                    fullPath,
                    function (texture) {
                        texture.colorSpace = THREE.SRGBColorSpace;
                        texture.anisotropy = options.anisotropy || 8;
                        if (options.wrapS !== undefined) {
                            texture.wrapS = options.wrapS;
                            texture.wrapT = options.wrapT !== undefined ? options.wrapT : options.wrapS;
                        }
                        if (options.repeat) {
                            texture.repeat.set(options.repeat.x, options.repeat.y);
                        }
                        self._cache.set(fullPath, texture);
                        resolve(texture);
                    },
                    undefined,
                    function () {
                        if (format !== '.png') {
                            var pngLoader = new THREE.TextureLoader();
                            var pngPath = basePath + '.png';
                            pngLoader.load(
                                pngPath,
                                function (texture) {
                                    texture.colorSpace = THREE.SRGBColorSpace;
                                    texture.anisotropy = options.anisotropy || 8;
                                    self._cache.set(fullPath, texture);
                                    resolve(texture);
                                },
                                undefined,
                                function () {
                                    reject(new Error('Failed to load texture: ' + fullPath));
                                }
                            );
                        } else {
                            reject(new Error('Failed to load texture: ' + fullPath));
                        }
                    }
                );
            }
        });
    };

    TextureLoader.prototype.clearCache = function () {
        this._cache.forEach(function (texture) {
            if (texture && typeof texture.dispose === 'function') {
                texture.dispose();
            }
        });
        this._cache.clear();
    };

    TextureLoader.prototype.getCacheStats = function () {
        return {
            count: this._cache.size,
            ktx2Supported: this._ktx2Supported,
            basisSupported: this._basisSupported
        };
    };

    global.TextureLoader = TextureLoader;

    if (typeof window !== 'undefined') {
        window.TextureLoader = TextureLoader;
    }

})(typeof window !== 'undefined' ? window : this);
