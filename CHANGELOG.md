# Changelog

## [0.4.0](https://github.com/garethgeorge/restora/compare/v0.3.0...v0.4.0) (2023-12-04)


### Features

* implement prune support ([#25](https://github.com/garethgeorge/restora/issues/25)) ([a311b0a](https://github.com/garethgeorge/restora/commit/a311b0a3fb5315f17d66361a3e72fa10b8a744a1))
* implement restore operation through snapshot browser UI ([#27](https://github.com/garethgeorge/restora/issues/27)) ([d758509](https://github.com/garethgeorge/restora/commit/d758509797e21e3ec4bc67eff4d974604e4a5476))

## [0.3.0](https://github.com/garethgeorge/restora/compare/v0.2.0...v0.3.0) (2023-12-03)


### Features

* autoinstall required restic version ([b385c01](https://github.com/garethgeorge/restora/commit/b385c011210087e6d6992a4e4b279fec4b22ab89))
* basic forget support in backend and UI ([d22d9d1](https://github.com/garethgeorge/restora/commit/d22d9d1a05831fae94ce397c0c73c6292d378cf5))
* begin UI integration with backend ([cccdd29](https://github.com/garethgeorge/restora/commit/cccdd297c15cd47268b2a1903e9624bdbca3dc68))
* display queued operations ([0c818bb](https://github.com/garethgeorge/restora/commit/0c818bb9452a944d8b1127e553142e1e60ed90af))
* forget soft-deletes operations associated with removed snapshots ([f3dc7ff](https://github.com/garethgeorge/restora/commit/f3dc7ffd077fef67870852f8f4e8b9aa6c94806e))
* forget soft-deletes operations associated with removed snapshots ([38bc107](https://github.com/garethgeorge/restora/commit/38bc107db394716e34245f1edefc5e4cf4a15333))
* implement add plan UI ([9288589](https://github.com/garethgeorge/restora/commit/92885898cf551a2dcb4bb315f130138cd7a8cc67))
* implement backup scheduling in orchestrator ([eadb1a8](https://github.com/garethgeorge/restora/commit/eadb1a82019f0cfc82edf8559adbad7730a4e86a))
* implement basic plan view ([4c6f042](https://github.com/garethgeorge/restora/commit/4c6f042250946a036e46225e669ee39e2433b198))
* implement delete button for plan and repo UIs ([ffb0d85](https://github.com/garethgeorge/restora/commit/ffb0d859f19f4af66a7521768dab083995f9672a))
* implement forget and prune support in restic pkg ([ffb4573](https://github.com/garethgeorge/restora/commit/ffb4573737a73cc32f325bc0b9c3feed764b7879))
* implement forget operation ([ebccf3b](https://github.com/garethgeorge/restora/commit/ebccf3bc3b78083aee635de7c6ae23b52ee88284))
* implement repo, edit, and supporting RPCs ([d282c32](https://github.com/garethgeorge/restora/commit/d282c32c8bd3d8f5747e934d4af6a84faca1ec86))
* implement snapshot browsing ([8ffffa0](https://github.com/garethgeorge/restora/commit/8ffffa05e41ca31e2d38fde5427dae34ac4a1abb))
* implement snapshot indexing ([a90b30e](https://github.com/garethgeorge/restora/commit/a90b30e19f7107874bbfe244451b07f72c437213))
* improve oplist performance and display forget operations in oplist ([#22](https://github.com/garethgeorge/restora/issues/22)) ([51b4921](https://github.com/garethgeorge/restora/commit/51b49214e3d32cc4b28e13085bd196ba164a8c19))
* initial oplog implementation ([dd9142c](https://github.com/garethgeorge/restora/commit/dd9142c0e97e1175ff12f2861220af0e0d68b7d9))
* initial optree implementation ([ba390a2](https://github.com/garethgeorge/restora/commit/ba390a2ca1b5e9adaab36a7db0d988f54f5a6cdd))
* operations IDs are ordered by operation timestamp ([a1ed6f9](https://github.com/garethgeorge/restora/commit/a1ed6f90ba1d608e00c53221db45b67251085aa7))
* present list of operations on plan view ([6491dbe](https://github.com/garethgeorge/restora/commit/6491dbed146967c0e12eee4392d1d12843dc7c5e))
* repo can be created through UI ([9ccade5](https://github.com/garethgeorge/restora/commit/9ccade5ccd97f4e485d52ad5c675be6b0a4a1049))
* scaffolding basic UI structure ([1273f81](https://github.com/garethgeorge/restora/commit/1273f8105a2549b0ccd0c7a588eb60646b66366e))
* show snapshots in sidenav ([1a9a5b6](https://github.com/garethgeorge/restora/commit/1a9a5b60d24dd75752e5a3f84dd87af3e38422bb))
* snapshot items are viewable in the UI and minor element ordering fixes ([a333001](https://github.com/garethgeorge/restora/commit/a33300175c645f31b95b3038de02821a1f3d5559))
* support ImportSnapshotOperation in oplog ([89f95b3](https://github.com/garethgeorge/restora/commit/89f95b351fe250534cd39ac27ff34b2b148256e1))
* update getting started guide ([2c421d6](https://github.com/garethgeorge/restora/commit/2c421d661501fa4a3120aa3f39937cd58b29c2dc))


### Bug Fixes

* build and test fixes ([4957496](https://github.com/garethgeorge/restora/commit/49574967871494dcb5095e5699610097466f57f9))
* connectivity issues with embedded server ([482cc8e](https://github.com/garethgeorge/restora/commit/482cc8ebbc93b919991f6566b212247c5874f70f))
* deadlock in snapshots ([93b2120](https://github.com/garethgeorge/restora/commit/93b2120f74ea348e5084ab430573368bf4066eec))
* forget deadlocking and misc smaller bugs ([b7c633d](https://github.com/garethgeorge/restora/commit/b7c633d021d68d4880a5f442ce70a858002b4af2))
* improve error message formatting ([ae33b01](https://github.com/garethgeorge/restora/commit/ae33b01de408af3b1d711a369298a2782a24ad1e))
* improve operation ordering to fix snapshots indexed before forget operation ([#21](https://github.com/garethgeorge/restora/issues/21)) ([b513b08](https://github.com/garethgeorge/restora/commit/b513b08e51434c28c90f5f062b4ae292f6854f4e))
* improve UI performance ([8488d46](https://github.com/garethgeorge/restora/commit/8488d461bd7ffec2e8171d67f83093c32c79073f))
* repo orchestrator tests ([d077fc8](https://github.com/garethgeorge/restora/commit/d077fc83c97b7fbdbeda9702828c8780182b2616))
* restic fails to detect summary event for very short backups ([46b2a85](https://github.com/garethgeorge/restora/commit/46b2a8567706ddb21cfcf3e18b57e16d50809b56))
* standardize on fully qualified snapshot_id and decouple protobufs from restic package ([e6031bf](https://github.com/garethgeorge/restora/commit/e6031bfa543a7300e622c1b0f56efc6320e7611e))
* support more versions of restic ([0cdfd11](https://github.com/garethgeorge/restora/commit/0cdfd115e29a0b08d5814e71c0f4a8f2baf52e90))
* task priority not taking effect ([af7462c](https://github.com/garethgeorge/restora/commit/af7462cefb130153cdaaa08e8ebefefa40e80e49))
* time formatting in operation list ([53c7f12](https://github.com/garethgeorge/restora/commit/53c7f1248f5284080fff872ac79b3996474412b3))
* UI layout adjustments ([7d1b95c](https://github.com/garethgeorge/restora/commit/7d1b95c81f0f69840ce1d20cb0d4a4bb90011dc9))

## [0.2.0](https://github.com/garethgeorge/restora/compare/v0.1.3...v0.2.0) (2023-12-03)


### Features

* forget soft-deletes operations associated with removed snapshots ([f3dc7ff](https://github.com/garethgeorge/restora/commit/f3dc7ffd077fef67870852f8f4e8b9aa6c94806e))
* forget soft-deletes operations associated with removed snapshots ([38bc107](https://github.com/garethgeorge/restora/commit/38bc107db394716e34245f1edefc5e4cf4a15333))
* improve oplist performance and display forget operations in oplist ([#22](https://github.com/garethgeorge/restora/issues/22)) ([51b4921](https://github.com/garethgeorge/restora/commit/51b49214e3d32cc4b28e13085bd196ba164a8c19))


### Bug Fixes

* forget deadlocking and misc smaller bugs ([b7c633d](https://github.com/garethgeorge/restora/commit/b7c633d021d68d4880a5f442ce70a858002b4af2))
* improve operation ordering to fix snapshots indexed before forget operation ([#21](https://github.com/garethgeorge/restora/issues/21)) ([b513b08](https://github.com/garethgeorge/restora/commit/b513b08e51434c28c90f5f062b4ae292f6854f4e))
* task priority not taking effect ([af7462c](https://github.com/garethgeorge/restora/commit/af7462cefb130153cdaaa08e8ebefefa40e80e49))
* UI layout adjustments ([7d1b95c](https://github.com/garethgeorge/restora/commit/7d1b95c81f0f69840ce1d20cb0d4a4bb90011dc9))

## [0.2.0](https://github.com/garethgeorge/restora/compare/v0.1.3...v0.2.0) (2023-12-02)


### Features

* forget soft-deletes operations associated with removed snapshots ([f3dc7ff](https://github.com/garethgeorge/restora/commit/f3dc7ffd077fef67870852f8f4e8b9aa6c94806e))
* forget soft-deletes operations associated with removed snapshots ([38bc107](https://github.com/garethgeorge/restora/commit/38bc107db394716e34245f1edefc5e4cf4a15333))


### Bug Fixes

* forget deadlocking and misc smaller bugs ([b7c633d](https://github.com/garethgeorge/restora/commit/b7c633d021d68d4880a5f442ce70a858002b4af2))
* improve operation ordering to fix snapshots indexed before forget operation ([#21](https://github.com/garethgeorge/restora/issues/21)) ([b513b08](https://github.com/garethgeorge/restora/commit/b513b08e51434c28c90f5f062b4ae292f6854f4e))
* task priority not taking effect ([af7462c](https://github.com/garethgeorge/restora/commit/af7462cefb130153cdaaa08e8ebefefa40e80e49))
