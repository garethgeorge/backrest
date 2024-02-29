# Changelog

## [0.14.0](https://github.com/garethgeorge/backrest/compare/v0.13.0...v0.14.0) (2024-02-29)


### Features

* add 'compute stats' button to refresh stats on repo view ([1f42b6a](https://github.com/garethgeorge/backrest/commit/1f42b6ab4e0313bbb12e6bc22b561d7544504644))
* add release artifacts for arm32 ([a737371](https://github.com/garethgeorge/backrest/commit/a737371ed559f5b65e734b0d97c44dcb2749ce53))
* check for basic auth ([#110](https://github.com/garethgeorge/backrest/issues/110)) ([#129](https://github.com/garethgeorge/backrest/issues/129)) ([871c54f](https://github.com/garethgeorge/backrest/commit/871c54f35f8651632714ca7d3a3ab0e809549b51))
* improved stats visualization with graphs and cleanup operation filtering ([5b362cc](https://github.com/garethgeorge/backrest/commit/5b362ccbb45e59954dad574b93848195d45b55ef))
* support flag overrides for 'restic backup' in plan configuration ([56f5e40](https://github.com/garethgeorge/backrest/commit/56f5e405037a6309a3d1299356b363cd84281aef))


### Bug Fixes

* alpine linux Dockerfile and add openssh ([3cb9d27](https://github.com/garethgeorge/backrest/commit/3cb9d2717c1bda7bb7ed4e029ac938c851b9f664))
* backrest shows hidden operations in list view ([c013f06](https://github.com/garethgeorge/backrest/commit/c013f069ff5eab6177d2bde373f23efe34b1aa8d))
* BackupInfoCollector handling of filtered events ([f1e4619](https://github.com/garethgeorge/backrest/commit/f1e4619e9d98416289fb0ee51d56ff48e163b85d))
* install rclone with apk for alpine image ([#138](https://github.com/garethgeorge/backrest/issues/138)) ([79715a9](https://github.com/garethgeorge/backrest/commit/79715a97b34af60ca90894065d89c9ae603f0a59))
* properly parse repo flags ([348ec46](https://github.com/garethgeorge/backrest/commit/348ec4690cab74c3089f2be33d889df3002a5a97))
* stat operation interval for long running repos ([f2477ab](https://github.com/garethgeorge/backrest/commit/f2477ab06cbe571723cd7290e06e8890747f81aa))

## [0.13.0](https://github.com/garethgeorge/backrest/compare/v0.12.2...v0.13.0) (2024-02-21)


### Features

* add case insensitive excludes (iexcludes) ([#108](https://github.com/garethgeorge/backrest/issues/108)) ([bf6fb7e](https://github.com/garethgeorge/backrest/commit/bf6fb7e71402590961271e91ad6da63db27ff5ad))
* add flags to configure backrest options e.g. --config-file, --data-dir, --restic-cmd, --bind-address ([41ddc8e](https://github.com/garethgeorge/backrest/commit/41ddc8e1a9d5501a92498c8cf3c72625bd181f8a))
* add opt-in auto-unlock feature to remove locks on forget and prune ([#107](https://github.com/garethgeorge/backrest/issues/107)) ([c1ee33f](https://github.com/garethgeorge/backrest/commit/c1ee33f0cd65a23ec0090852ee0fc5fa50e72b64))
* add rclone binary to docker image and arm64 support ([#105](https://github.com/garethgeorge/backrest/issues/105)) ([5a49f2f](https://github.com/garethgeorge/backrest/commit/5a49f2f063e887cba85bba0347ebce3efe15753e))
* bundle rclone, busybox commands, and bash in default backrest docker image ([cec04f8](https://github.com/garethgeorge/backrest/commit/cec04f8f745d4bcfd49829c43367c61cb9778174))
* display non-fatal errors in backup operations (e.g. unreadable files) in UI ([#100](https://github.com/garethgeorge/backrest/issues/100)) ([caac35a](https://github.com/garethgeorge/backrest/commit/caac35a5402d056b626b59d19084d6a699d4346d))


### Bug Fixes

* improve error message when rclone config is missing ([663b430](https://github.com/garethgeorge/backrest/commit/663b430598e0890df74989af12ae81fae7922251))
* improved sidebar status refresh interval during live operations ([3d192fd](https://github.com/garethgeorge/backrest/commit/3d192fd59d98c242ed583d00eeec37e68a4a2ff5))
* live backup progress updates with partial-backup errors ([97a4948](https://github.com/garethgeorge/backrest/commit/97a494847ac5031866c31db0bb32219e6b2a0038))
* migrate prune policy options to oneof ([ef41d34](https://github.com/garethgeorge/backrest/commit/ef41d34d5312b6a3bcc4af536f64275cd20da657))
* restore operations should succeed for unassociated snapshots ([448107d](https://github.com/garethgeorge/backrest/commit/448107d22612f040fd45493246088277a4a72f63))
* separate docker images for scratch and alpine linux base ([#106](https://github.com/garethgeorge/backrest/issues/106)) ([40e3e04](https://github.com/garethgeorge/backrest/commit/40e3e04a686f0a1749fa39e15821e6310e0ccf52))

## [0.12.2](https://github.com/garethgeorge/backrest/compare/v0.12.1...v0.12.2) (2024-02-16)


### Bug Fixes

* release-please automation ([63ddf15](https://github.com/garethgeorge/backrest/commit/63ddf15bf9799de30bda8548421e11e1bcd9ed05))

## [0.12.1](https://github.com/garethgeorge/backrest/compare/v0.12.0...v0.12.1) (2024-02-16)


### Bug Fixes

* delete event button in UI is hard to see on light theme ([8a05df8](https://github.com/garethgeorge/backrest/commit/8a05df87fcc44699c890f0cbe1065d79f49e1cc2))
* use 'embed' to package WebUI sources instead of go.rice ([e3ba5cf](https://github.com/garethgeorge/backrest/commit/e3ba5cf12ebfedafaa2125687bd7522f29ccab51))

## [0.12.0](https://github.com/garethgeorge/backrest/compare/v0.11.1...v0.12.0) (2024-02-15)


### Features

* add button to forget individual snapshots ([276b1d2](https://github.com/garethgeorge/backrest/commit/276b1d2c602ad0f787958452070771af3e69f073))
* add slack webhook ([8fa90ab](https://github.com/garethgeorge/backrest/commit/8fa90ab9ca48f0888ed0a5d263cb697758063188))
* Add support for multiple sets of expected env vars per repo scheme ([#90](https://github.com/garethgeorge/backrest/issues/90)) ([da0551c](https://github.com/garethgeorge/backrest/commit/da0551c19a98fe675d278e34f8e3cc58ac9edaf5))
* clear operations from history ([dc7a3a5](https://github.com/garethgeorge/backrest/commit/dc7a3a59a2400f97dd6b8140c6e70a34105496f9))
* Windows WebUI uses correct path separator ([f5521e7](https://github.com/garethgeorge/backrest/commit/f5521e7b56e446fa2062a95560f315621b77d3e6))


### Bug Fixes

* cleanup old versions of restic when upgrading ([79f529f](https://github.com/garethgeorge/backrest/commit/79f529f8edfb9bf893e74f7b1355bd7f2d7bdc3f))
* hide delete operation button if operation is in progress or pending ([08c8762](https://github.com/garethgeorge/backrest/commit/08c876243febb99a68740c449055e850f37d740e))
* retention policy configuration in add plan view ([dd24d90](https://github.com/garethgeorge/backrest/commit/dd24d9024f5ade62535956b1449dae75627ce493))
* stats operations running at wrong interval ([05e5ae0](https://github.com/garethgeorge/backrest/commit/05e5ae0c455680bf9fbc9b4b2a9fbf96bcfdfc3b))

## [0.11.1](https://github.com/garethgeorge/backrest/compare/v0.11.0...v0.11.1) (2024-02-08)


### Bug Fixes

* backrest fails to create directory for jwt secrets ([0067edf](https://github.com/garethgeorge/backrest/commit/0067edf378b01147f0041c225994098cb9c452ab))
* form bugs in UI e.g. awkward behavior when modifying hooks ([4fcf526](https://github.com/garethgeorge/backrest/commit/4fcf52602c114e2c639fc4302a9b8e8d51180a4d))
* update restic version to 1.16.4 ([668a7cb](https://github.com/garethgeorge/backrest/commit/668a7cb5bb5c0955a0e3186b2dd9329cedddd96f))
* wrong field names in hooks form ([3540904](https://github.com/garethgeorge/backrest/commit/354090497b73d40d8a9e705d1aa0c4662ffc4b0e))
* wrong value passed to --max-unused when providing a custom prune policy ([34175f2](https://github.com/garethgeorge/backrest/commit/34175f273630f7d2324a4d6b5f9f2f7576dd6608))

## [0.11.0](https://github.com/garethgeorge/backrest/compare/v0.10.1...v0.11.0) (2024-02-04)


### Features

* add user configurable command hooks for backup lifecycle events ([#60](https://github.com/garethgeorge/backrest/issues/60)) ([9be413b](https://github.com/garethgeorge/backrest/commit/9be413bbcca796862f161a769991ab695a50b464))
* authentication for WebUI ([#62](https://github.com/garethgeorge/backrest/issues/62)) ([4a1f326](https://github.com/garethgeorge/backrest/commit/4a1f3268a7de0533e0a979b9e97a7117b028358e))
* implement discord hook type ([25924b6](https://github.com/garethgeorge/backrest/commit/25924b6197c870f9dfc1e04f5be39377251e7f2d))
* implement gotify hook type ([e0ce655](https://github.com/garethgeorge/backrest/commit/e0ce6558c047f3aff068ee5d475fa1bdba380c4d))
* support keep-all retention policy for append-only backups ([f163c02](https://github.com/garethgeorge/backrest/commit/f163c02d7d2c798b4057037a996de44e34de9f2b))


### Bug Fixes

* add API test coverage and fix minor bugs ([f5bb74b](https://github.com/garethgeorge/backrest/commit/f5bb74bf246fcd5712dbbc85f4233169f7db4aa7))
* add first time setup hint for user authentication ([4a565f2](https://github.com/garethgeorge/backrest/commit/4a565f2cdcd091e0eabc302ab91e53012f53eb26))
* add test coverage for log rotation ([f1084ca](https://github.com/garethgeorge/backrest/commit/f1084cab4894751ba4a92f9be6b6b70d9084d0e6))
* bugfixes for auth flow ([427792c](https://github.com/garethgeorge/backrest/commit/427792c7244fb712bbea0557d4a6c7ee07052534))
* stats not displaying on long running repos ([f1ba1d9](https://github.com/garethgeorge/backrest/commit/f1ba1d91f37234f24ae5202d27114a33432366da))
* store large log outputs in tar bundles of logs ([0cf01e0](https://github.com/garethgeorge/backrest/commit/0cf01e020640b0145bcd0d25a38cde1fce940aff))
* windows install errors on decompressing zip archive ([5323b9f](https://github.com/garethgeorge/backrest/commit/5323b9ffc065bc3b28171575cdccc4358b69750b))

## [0.10.1](https://github.com/garethgeorge/backrest/compare/v0.10.0...v0.10.1) (2024-01-25)


### Bug Fixes

* chmod config 0600 such that only the creating user can read ([ecff0e5](https://github.com/garethgeorge/backrest/commit/ecff0e57c1fa4d65f35774d227a27222af8e7921))
* install scripts handle working dir correctly ([dcff2ad](https://github.com/garethgeorge/backrest/commit/dcff2adf60222030043d7a227d27e74f555ab376))
* relax name regex for plans and repos ([ee6134a](https://github.com/garethgeorge/backrest/commit/ee6134af76c3e90f542f67b89b2571f060db5590))
* sftp support using public key authentication ([bedb302](https://github.com/garethgeorge/backrest/commit/bedb302a025438a58309f26b046c9b6d49316414))
* typos in validation error messages in addrepomodel ([3b79afb](https://github.com/garethgeorge/backrest/commit/3b79afb2b18530deaa10cca08a60941a64c6fd9b))

## [0.10.0](https://github.com/garethgeorge/backrest/compare/v0.9.3...v0.10.0) (2024-01-15)


### Features

* make prune policy configurable in the addrepoview in the UI ([3fd08eb](https://github.com/garethgeorge/backrest/commit/3fd08eb8e4b455db656a0680318851824fdad2db))
* update restic dependency to v0.16.3 ([ac8db31](https://github.com/garethgeorge/backrest/commit/ac8db31713d4db3c2240b7f7c006e518e9e0726c))
* verify gpg signature when downloading and installing restic binary ([04106d1](https://github.com/garethgeorge/backrest/commit/04106d15d5ad73db6e670e84340ac1f9be200a23))

## [0.9.3](https://github.com/garethgeorge/backrest/compare/v0.9.2...v0.9.3) (2024-01-05)


### Bug Fixes

* correctly mark tasks as inprogress before execution ([b19438a](https://github.com/garethgeorge/backrest/commit/b19438afbd7b83dc964774347e64491143a3a5d2))
* correctly select light/dark mode based on system colortheme ([b64199c](https://github.com/garethgeorge/backrest/commit/b64199c140db7d2a77b58219cee088d22ec81954))

## [0.9.2](https://github.com/garethgeorge/backrest/compare/v0.9.1...v0.9.2) (2024-01-01)


### Bug Fixes

* possible race condition in scheduled task heap ([30874c9](https://github.com/garethgeorge/backrest/commit/30874c9150f32a0fba5f1ea99bc77bcc978d8b03))

## [0.9.1](https://github.com/garethgeorge/backrest/compare/v0.9.0...v0.9.1) (2024-01-01)


### Bug Fixes

* failed forget operations are hidden in the UI ([9896446](https://github.com/garethgeorge/backrest/commit/9896446ccfbcb8475a21b5fb565ebb73cb6bac2c))
* UI buttons spin while waiting for tasks to complete ([c767fa7](https://github.com/garethgeorge/backrest/commit/c767fa7476d76f1b4eb49443a19ee1cedb4eb70a))

## [0.9.0](https://github.com/garethgeorge/backrest/compare/v0.8.1...v0.9.0) (2023-12-31)


### Features

* add backrest logo ([5add0d8](https://github.com/garethgeorge/backrest/commit/5add0d8ffa829a71103520c94eacae17966f2a9f))
* add mobile layout ([9c7f227](https://github.com/garethgeorge/backrest/commit/9c7f227ad0f5df34d66390c94b64e9f5181d24f0))
* index snapshots created outside of backrest ([7711297](https://github.com/garethgeorge/backrest/commit/7711297a84170a733c5ccdb3e89617efc878cf69))
* schedule index operations and stats refresh from repo view ([851bd12](https://github.com/garethgeorge/backrest/commit/851bd125b640e65a5b98b67d28d2f29e94411646))


### Bug Fixes

* operations associated with incorrect ID when tasks are rescheduled ([25871c9](https://github.com/garethgeorge/backrest/commit/25871c99920d8717e91bf1a921109b9df82a59a1))
* reduce stats refresh frequency ([adbe005](https://github.com/garethgeorge/backrest/commit/adbe0056d82a5d9f890ce79b1120f5084bdc7124))
* stat never runs ([3f3252d](https://github.com/garethgeorge/backrest/commit/3f3252d47951270fbf5f21b0831effb121d3ba3f))
* stats task priority ([6bfe769](https://github.com/garethgeorge/backrest/commit/6bfe769fe037a5f2d35947574a5ed7e26ba981a8))
* tasks run late when laptops resume from sleep ([cb78298](https://github.com/garethgeorge/backrest/commit/cb78298cffb492560717d5f8bdcd5941f7976f2e))
* UI and code quality improvements ([c5e435d](https://github.com/garethgeorge/backrest/commit/c5e435d640bc8e79ceacf7f64d4cf75644859204))

## [0.8.0](https://github.com/garethgeorge/backrest/compare/v0.7.0...v0.8.0) (2023-12-25)


### Features

* add repo stats to restic package ([26d4724](https://github.com/garethgeorge/backrest/commit/26d47243c1e31f17c4d8adc6227325551854ce1f))
* add stats to repo view e.g. total size in storage ([adb0e3f](https://github.com/garethgeorge/backrest/commit/adb0e3f23050a86cd1c507d374e9d45f5eb5ee27))
* display last operation status for each plan and repo in UI ([cc11197](https://github.com/garethgeorge/backrest/commit/cc111970ca2e61cf39804378808aa5b5f77f9581))


### Bug Fixes

* crashing bug on partial backup ([#39](https://github.com/garethgeorge/backrest/issues/39)) ([fba6c8d](https://github.com/garethgeorge/backrest/commit/fba6c8da869d66b7b44f87a0dc1e3779924c31b7))
* install scripts and improved asset compression ([b8c2e81](https://github.com/garethgeorge/backrest/commit/b8c2e813586f2b48c78d70e09a29c5052621caf1))

## [0.7.0](https://github.com/garethgeorge/backrest/compare/v0.6.0...v0.7.0) (2023-12-22)


### Features

* add activity bar to UI heading ([f5c3e76](https://github.com/garethgeorge/backrest/commit/f5c3e762ed4ed3c908e843d74985fb6c7b253db7))
* add clear error history button ([48d80b9](https://github.com/garethgeorge/backrest/commit/48d80b9473db6619518924d0849b0eda78e62afa))
* add repo view ([9522ac1](https://github.com/garethgeorge/backrest/commit/9522ac18deedc15311d3d464ee36c20e7f72e39f))
* autoinstall required restic version ([b385c01](https://github.com/garethgeorge/backrest/commit/b385c011210087e6d6992a4e4b279fec4b22ab89))
* basic forget support in backend and UI ([d22d9d1](https://github.com/garethgeorge/backrest/commit/d22d9d1a05831fae94ce397c0c73c6292d378cf5))
* begin UI integration with backend ([cccdd29](https://github.com/garethgeorge/backrest/commit/cccdd297c15cd47268b2a1903e9624bdbca3dc68))
* display queued operations ([0c818bb](https://github.com/garethgeorge/backrest/commit/0c818bb9452a944d8b1127e553142e1e60ed90af))
* forget soft-deletes operations associated with removed snapshots ([f3dc7ff](https://github.com/garethgeorge/backrest/commit/f3dc7ffd077fef67870852f8f4e8b9aa6c94806e))
* forget soft-deletes operations associated with removed snapshots ([38bc107](https://github.com/garethgeorge/backrest/commit/38bc107db394716e34245f1edefc5e4cf4a15333))
* implement add plan UI ([9288589](https://github.com/garethgeorge/backrest/commit/92885898cf551a2dcb4bb315f130138cd7a8cc67))
* implement backup scheduling in orchestrator ([eadb1a8](https://github.com/garethgeorge/backrest/commit/eadb1a82019f0cfc82edf8559adbad7730a4e86a))
* implement basic plan view ([4c6f042](https://github.com/garethgeorge/backrest/commit/4c6f042250946a036e46225e669ee39e2433b198))
* implement delete button for plan and repo UIs ([ffb0d85](https://github.com/garethgeorge/backrest/commit/ffb0d859f19f4af66a7521768dab083995f9672a))
* implement forget and prune support in restic pkg ([ffb4573](https://github.com/garethgeorge/backrest/commit/ffb4573737a73cc32f325bc0b9c3feed764b7879))
* implement forget operation ([ebccf3b](https://github.com/garethgeorge/backrest/commit/ebccf3bc3b78083aee635de7c6ae23b52ee88284))
* implement garbage collection of old operations ([46456a8](https://github.com/garethgeorge/backrest/commit/46456a88870934506ede4b67c3dfaa2f2afcee14))
* implement prune support ([#25](https://github.com/garethgeorge/backrest/issues/25)) ([a311b0a](https://github.com/garethgeorge/backrest/commit/a311b0a3fb5315f17d66361a3e72fa10b8a744a1))
* implement repo unlocking and operation list implementation ([6665ad9](https://github.com/garethgeorge/backrest/commit/6665ad98d7f54bea30ea532932a8a3409717c913))
* implement repo, edit, and supporting RPCs ([d282c32](https://github.com/garethgeorge/backrest/commit/d282c32c8bd3d8f5747e934d4af6a84faca1ec86))
* implement restore operation through snapshot browser UI ([#27](https://github.com/garethgeorge/backrest/issues/27)) ([d758509](https://github.com/garethgeorge/backrest/commit/d758509797e21e3ec4bc67eff4d974604e4a5476))
* implement snapshot browsing ([8ffffa0](https://github.com/garethgeorge/backrest/commit/8ffffa05e41ca31e2d38fde5427dae34ac4a1abb))
* implement snapshot indexing ([a90b30e](https://github.com/garethgeorge/backrest/commit/a90b30e19f7107874bbfe244451b07f72c437213))
* improve oplist performance and display forget operations in oplist ([#22](https://github.com/garethgeorge/backrest/issues/22)) ([51b4921](https://github.com/garethgeorge/backrest/commit/51b49214e3d32cc4b28e13085bd196ba164a8c19))
* initial oplog implementation ([dd9142c](https://github.com/garethgeorge/backrest/commit/dd9142c0e97e1175ff12f2861220af0e0d68b7d9))
* initial optree implementation ([ba390a2](https://github.com/garethgeorge/backrest/commit/ba390a2ca1b5e9adaab36a7db0d988f54f5a6cdd))
* initial Windows OS support ([f048cbf](https://github.com/garethgeorge/backrest/commit/f048cbf10dc60da51cd7f5aee4614a8750fd85b2))
* match system color theme (darkmode support) ([a8762dc](https://github.com/garethgeorge/backrest/commit/a8762dca329927b93db40b01cc011c00e12891f0))
* operations IDs are ordered by operation timestamp ([a1ed6f9](https://github.com/garethgeorge/backrest/commit/a1ed6f90ba1d608e00c53221db45b67251085aa7))
* present list of operations on plan view ([6491dbe](https://github.com/garethgeorge/backrest/commit/6491dbed146967c0e12eee4392d1d12843dc7c5e))
* repo can be created through UI ([9ccade5](https://github.com/garethgeorge/backrest/commit/9ccade5ccd97f4e485d52ad5c675be6b0a4a1049))
* scaffolding basic UI structure ([1273f81](https://github.com/garethgeorge/backrest/commit/1273f8105a2549b0ccd0c7a588eb60646b66366e))
* show snapshots in sidenav ([1a9a5b6](https://github.com/garethgeorge/backrest/commit/1a9a5b60d24dd75752e5a3f84dd87af3e38422bb))
* snapshot items are viewable in the UI and minor element ordering fixes ([a333001](https://github.com/garethgeorge/backrest/commit/a33300175c645f31b95b3038de02821a1f3d5559))
* support ImportSnapshotOperation in oplog ([89f95b3](https://github.com/garethgeorge/backrest/commit/89f95b351fe250534cd39ac27ff34b2b148256e1))
* support task cancellation ([fc9c06d](https://github.com/garethgeorge/backrest/commit/fc9c06df00409b73dda23f4be031746f492b1a24))
* update getting started guide ([2c421d6](https://github.com/garethgeorge/backrest/commit/2c421d661501fa4a3120aa3f39937cd58b29c2dc))


### Bug Fixes

* backup ordering in tree view ([b9c8b3e](https://github.com/garethgeorge/backrest/commit/b9c8b3e378e88a0feff4d477d9d97eb5db075382))
* build and test fixes ([4957496](https://github.com/garethgeorge/backrest/commit/49574967871494dcb5095e5699610097466f57f9))
* connectivity issues with embedded server ([482cc8e](https://github.com/garethgeorge/backrest/commit/482cc8ebbc93b919991f6566b212247c5874f70f))
* deadlock in snapshots ([93b2120](https://github.com/garethgeorge/backrest/commit/93b2120f74ea348e5084ab430573368bf4066eec))
* forget deadlocking and misc smaller bugs ([b7c633d](https://github.com/garethgeorge/backrest/commit/b7c633d021d68d4880a5f442ce70a858002b4af2))
* hide no-op prune operations ([20dd78c](https://github.com/garethgeorge/backrest/commit/20dd78cac4bdd6385cb7a0ea9ff0be75fde9135b))
* improve error message formatting ([ae33b01](https://github.com/garethgeorge/backrest/commit/ae33b01de408af3b1d711a369298a2782a24ad1e))
* improve operation ordering to fix snapshots indexed before forget operation ([#21](https://github.com/garethgeorge/backrest/issues/21)) ([b513b08](https://github.com/garethgeorge/backrest/commit/b513b08e51434c28c90f5f062b4ae292f6854f4e))
* improve output detail collection for command failures ([c492f9b](https://github.com/garethgeorge/backrest/commit/c492f9ba63169942509349797ebe951879b53635))
* improve UI performance ([8488d46](https://github.com/garethgeorge/backrest/commit/8488d461bd7ffec2e8171d67f83093c32c79073f))
* improve Windows path handling ([426aad4](https://github.com/garethgeorge/backrest/commit/426aad4890d2de5d70cd2e0232c0d11c42606c92))
* incorrrect handling of oplog events in UI ([95ca96a](https://github.com/garethgeorge/backrest/commit/95ca96a31f2e1ead2702164ec8675e4b4f54cf1d))
* operation ordering in tree view ([2b4e1a2](https://github.com/garethgeorge/backrest/commit/2b4e1a2fdbf11b010ddbcd0b6fd2640d01e4dbc8))
* operations marked as 'warn' rather than 'error' for partial backups ([fe92b62](https://github.com/garethgeorge/backrest/commit/fe92b625780481193e0ab63fbbdddb889bbda2a8))
* ordering of operations when viewed from backup tree ([063f086](https://github.com/garethgeorge/backrest/commit/063f086a6e31df250dd9be42cdb5fa549307106f))
* race condition in snapshot browser ([f239b91](https://github.com/garethgeorge/backrest/commit/f239b9170415e063ec8d60a5b5e14ae3610b9bad))
* relax output parsing to skip over warnings ([8f85b74](https://github.com/garethgeorge/backrest/commit/8f85b747f57844bbc898668723eec50a1666aa39))
* repo orchestrator tests ([d077fc8](https://github.com/garethgeorge/backrest/commit/d077fc83c97b7fbdbeda9702828c8780182b2616))
* restic fails to detect summary event for very short backups ([46b2a85](https://github.com/garethgeorge/backrest/commit/46b2a8567706ddb21cfcf3e18b57e16d50809b56))
* restic should initialize repo on backup operation ([e57abbd](https://github.com/garethgeorge/backrest/commit/e57abbdcb1864c362e6ae3c22850c0380671cb34))
* restora should not init repos added manually e.g. without the UI ([68b50e1](https://github.com/garethgeorge/backrest/commit/68b50e1eb5a2ebd861c869f71f49d196cb5214f8))
* snapshots out of order in UI ([b9bcc7e](https://github.com/garethgeorge/backrest/commit/b9bcc7e7c758abafa4878b6ef895adf2d2d0bc42))
* standardize on fully qualified snapshot_id and decouple protobufs from restic package ([e6031bf](https://github.com/garethgeorge/backrest/commit/e6031bfa543a7300e622c1b0f56efc6320e7611e))
* support more versions of restic ([0cdfd11](https://github.com/garethgeorge/backrest/commit/0cdfd115e29a0b08d5814e71c0f4a8f2baf52e90))
* task cancellation ([d49b729](https://github.com/garethgeorge/backrest/commit/d49b72996ea7fd0543d55db3fc8e1127fe5a2476))
* task priority not taking effect ([af7462c](https://github.com/garethgeorge/backrest/commit/af7462cefb130153cdaaa08e8ebefefa40e80e49))
* time formatting in operation list ([53c7f12](https://github.com/garethgeorge/backrest/commit/53c7f1248f5284080fff872ac79b3996474412b3))
* UI layout adjustments ([7d1b95c](https://github.com/garethgeorge/backrest/commit/7d1b95c81f0f69840ce1d20cb0d4a4bb90011dc9))
* unexpected config location on MacOS ([8d40576](https://github.com/garethgeorge/backrest/commit/8d40576c6526d6f180c96fbeb81d7f59f56b51b8))
* use timezone offset when grouping operations in OperationTree ([06240bd](https://github.com/garethgeorge/backrest/commit/06240bd7adabd76424025030cfde2fb5e54c219f))

## [0.6.0](https://github.com/garethgeorge/backrest/compare/v0.5.0...v0.6.0) (2023-12-22)


### Features

* add activity bar to UI heading ([f5c3e76](https://github.com/garethgeorge/backrest/commit/f5c3e762ed4ed3c908e843d74985fb6c7b253db7))
* add clear error history button ([48d80b9](https://github.com/garethgeorge/backrest/commit/48d80b9473db6619518924d0849b0eda78e62afa))
* add repo view ([9522ac1](https://github.com/garethgeorge/backrest/commit/9522ac18deedc15311d3d464ee36c20e7f72e39f))
* implement garbage collection of old operations ([46456a8](https://github.com/garethgeorge/backrest/commit/46456a88870934506ede4b67c3dfaa2f2afcee14))
* support task cancellation ([fc9c06d](https://github.com/garethgeorge/backrest/commit/fc9c06df00409b73dda23f4be031746f492b1a24))


### Bug Fixes

* backup ordering in tree view ([b9c8b3e](https://github.com/garethgeorge/backrest/commit/b9c8b3e378e88a0feff4d477d9d97eb5db075382))
* hide no-op prune operations ([20dd78c](https://github.com/garethgeorge/backrest/commit/20dd78cac4bdd6385cb7a0ea9ff0be75fde9135b))
* incorrrect handling of oplog events in UI ([95ca96a](https://github.com/garethgeorge/backrest/commit/95ca96a31f2e1ead2702164ec8675e4b4f54cf1d))
* operation ordering in tree view ([2b4e1a2](https://github.com/garethgeorge/backrest/commit/2b4e1a2fdbf11b010ddbcd0b6fd2640d01e4dbc8))
* operations marked as 'warn' rather than 'error' for partial backups ([fe92b62](https://github.com/garethgeorge/backrest/commit/fe92b625780481193e0ab63fbbdddb889bbda2a8))
* race condition in snapshot browser ([f239b91](https://github.com/garethgeorge/backrest/commit/f239b9170415e063ec8d60a5b5e14ae3610b9bad))
* restic should initialize repo on backup operation ([e57abbd](https://github.com/garethgeorge/backrest/commit/e57abbdcb1864c362e6ae3c22850c0380671cb34))
* backrest should not init repos added manually e.g. without the UI ([68b50e1](https://github.com/garethgeorge/backrest/commit/68b50e1eb5a2ebd861c869f71f49d196cb5214f8))
* task cancellation ([d49b729](https://github.com/garethgeorge/backrest/commit/d49b72996ea7fd0543d55db3fc8e1127fe5a2476))
* use timezone offset when grouping operations in OperationTree ([06240bd](https://github.com/garethgeorge/backrest/commit/06240bd7adabd76424025030cfde2fb5e54c219f))

## [0.5.0](https://github.com/garethgeorge/backrest/compare/v0.4.0...v0.5.0) (2023-12-10)


### Features

* implement repo unlocking and operation list implementation ([6665ad9](https://github.com/garethgeorge/backrest/commit/6665ad98d7f54bea30ea532932a8a3409717c913))
* initial Windows OS support ([f048cbf](https://github.com/garethgeorge/backrest/commit/f048cbf10dc60da51cd7f5aee4614a8750fd85b2))
* match system color theme (darkmode support) ([a8762dc](https://github.com/garethgeorge/backrest/commit/a8762dca329927b93db40b01cc011c00e12891f0))


### Bug Fixes

* improve output detail collection for command failures ([c492f9b](https://github.com/garethgeorge/backrest/commit/c492f9ba63169942509349797ebe951879b53635))
* improve Windows path handling ([426aad4](https://github.com/garethgeorge/backrest/commit/426aad4890d2de5d70cd2e0232c0d11c42606c92))
* ordering of operations when viewed from backup tree ([063f086](https://github.com/garethgeorge/backrest/commit/063f086a6e31df250dd9be42cdb5fa549307106f))
* relax output parsing to skip over warnings ([8f85b74](https://github.com/garethgeorge/backrest/commit/8f85b747f57844bbc898668723eec50a1666aa39))
* snapshots out of order in UI ([b9bcc7e](https://github.com/garethgeorge/backrest/commit/b9bcc7e7c758abafa4878b6ef895adf2d2d0bc42))
* unexpected config location on MacOS ([8d40576](https://github.com/garethgeorge/backrest/commit/8d40576c6526d6f180c96fbeb81d7f59f56b51b8))

## [0.4.0](https://github.com/garethgeorge/backrest/compare/v0.3.0...v0.4.0) (2023-12-04)


### Features

* implement prune support ([#25](https://github.com/garethgeorge/backrest/issues/25)) ([a311b0a](https://github.com/garethgeorge/backrest/commit/a311b0a3fb5315f17d66361a3e72fa10b8a744a1))
* implement restore operation through snapshot browser UI ([#27](https://github.com/garethgeorge/backrest/issues/27)) ([d758509](https://github.com/garethgeorge/backrest/commit/d758509797e21e3ec4bc67eff4d974604e4a5476))

## [0.3.0](https://github.com/garethgeorge/backrest/compare/v0.2.0...v0.3.0) (2023-12-03)


### Features

* autoinstall required restic version ([b385c01](https://github.com/garethgeorge/backrest/commit/b385c011210087e6d6992a4e4b279fec4b22ab89))
* basic forget support in backend and UI ([d22d9d1](https://github.com/garethgeorge/backrest/commit/d22d9d1a05831fae94ce397c0c73c6292d378cf5))
* begin UI integration with backend ([cccdd29](https://github.com/garethgeorge/backrest/commit/cccdd297c15cd47268b2a1903e9624bdbca3dc68))
* display queued operations ([0c818bb](https://github.com/garethgeorge/backrest/commit/0c818bb9452a944d8b1127e553142e1e60ed90af))
* forget soft-deletes operations associated with removed snapshots ([f3dc7ff](https://github.com/garethgeorge/backrest/commit/f3dc7ffd077fef67870852f8f4e8b9aa6c94806e))
* forget soft-deletes operations associated with removed snapshots ([38bc107](https://github.com/garethgeorge/backrest/commit/38bc107db394716e34245f1edefc5e4cf4a15333))
* implement add plan UI ([9288589](https://github.com/garethgeorge/backrest/commit/92885898cf551a2dcb4bb315f130138cd7a8cc67))
* implement backup scheduling in orchestrator ([eadb1a8](https://github.com/garethgeorge/backrest/commit/eadb1a82019f0cfc82edf8559adbad7730a4e86a))
* implement basic plan view ([4c6f042](https://github.com/garethgeorge/backrest/commit/4c6f042250946a036e46225e669ee39e2433b198))
* implement delete button for plan and repo UIs ([ffb0d85](https://github.com/garethgeorge/backrest/commit/ffb0d859f19f4af66a7521768dab083995f9672a))
* implement forget and prune support in restic pkg ([ffb4573](https://github.com/garethgeorge/backrest/commit/ffb4573737a73cc32f325bc0b9c3feed764b7879))
* implement forget operation ([ebccf3b](https://github.com/garethgeorge/backrest/commit/ebccf3bc3b78083aee635de7c6ae23b52ee88284))
* implement repo, edit, and supporting RPCs ([d282c32](https://github.com/garethgeorge/backrest/commit/d282c32c8bd3d8f5747e934d4af6a84faca1ec86))
* implement snapshot browsing ([8ffffa0](https://github.com/garethgeorge/backrest/commit/8ffffa05e41ca31e2d38fde5427dae34ac4a1abb))
* implement snapshot indexing ([a90b30e](https://github.com/garethgeorge/backrest/commit/a90b30e19f7107874bbfe244451b07f72c437213))
* improve oplist performance and display forget operations in oplist ([#22](https://github.com/garethgeorge/backrest/issues/22)) ([51b4921](https://github.com/garethgeorge/backrest/commit/51b49214e3d32cc4b28e13085bd196ba164a8c19))
* initial oplog implementation ([dd9142c](https://github.com/garethgeorge/backrest/commit/dd9142c0e97e1175ff12f2861220af0e0d68b7d9))
* initial optree implementation ([ba390a2](https://github.com/garethgeorge/backrest/commit/ba390a2ca1b5e9adaab36a7db0d988f54f5a6cdd))
* operations IDs are ordered by operation timestamp ([a1ed6f9](https://github.com/garethgeorge/backrest/commit/a1ed6f90ba1d608e00c53221db45b67251085aa7))
* present list of operations on plan view ([6491dbe](https://github.com/garethgeorge/backrest/commit/6491dbed146967c0e12eee4392d1d12843dc7c5e))
* repo can be created through UI ([9ccade5](https://github.com/garethgeorge/backrest/commit/9ccade5ccd97f4e485d52ad5c675be6b0a4a1049))
* scaffolding basic UI structure ([1273f81](https://github.com/garethgeorge/backrest/commit/1273f8105a2549b0ccd0c7a588eb60646b66366e))
* show snapshots in sidenav ([1a9a5b6](https://github.com/garethgeorge/backrest/commit/1a9a5b60d24dd75752e5a3f84dd87af3e38422bb))
* snapshot items are viewable in the UI and minor element ordering fixes ([a333001](https://github.com/garethgeorge/backrest/commit/a33300175c645f31b95b3038de02821a1f3d5559))
* support ImportSnapshotOperation in oplog ([89f95b3](https://github.com/garethgeorge/backrest/commit/89f95b351fe250534cd39ac27ff34b2b148256e1))
* update getting started guide ([2c421d6](https://github.com/garethgeorge/backrest/commit/2c421d661501fa4a3120aa3f39937cd58b29c2dc))


### Bug Fixes

* build and test fixes ([4957496](https://github.com/garethgeorge/backrest/commit/49574967871494dcb5095e5699610097466f57f9))
* connectivity issues with embedded server ([482cc8e](https://github.com/garethgeorge/backrest/commit/482cc8ebbc93b919991f6566b212247c5874f70f))
* deadlock in snapshots ([93b2120](https://github.com/garethgeorge/backrest/commit/93b2120f74ea348e5084ab430573368bf4066eec))
* forget deadlocking and misc smaller bugs ([b7c633d](https://github.com/garethgeorge/backrest/commit/b7c633d021d68d4880a5f442ce70a858002b4af2))
* improve error message formatting ([ae33b01](https://github.com/garethgeorge/backrest/commit/ae33b01de408af3b1d711a369298a2782a24ad1e))
* improve operation ordering to fix snapshots indexed before forget operation ([#21](https://github.com/garethgeorge/backrest/issues/21)) ([b513b08](https://github.com/garethgeorge/backrest/commit/b513b08e51434c28c90f5f062b4ae292f6854f4e))
* improve UI performance ([8488d46](https://github.com/garethgeorge/backrest/commit/8488d461bd7ffec2e8171d67f83093c32c79073f))
* repo orchestrator tests ([d077fc8](https://github.com/garethgeorge/backrest/commit/d077fc83c97b7fbdbeda9702828c8780182b2616))
* restic fails to detect summary event for very short backups ([46b2a85](https://github.com/garethgeorge/backrest/commit/46b2a8567706ddb21cfcf3e18b57e16d50809b56))
* standardize on fully qualified snapshot_id and decouple protobufs from restic package ([e6031bf](https://github.com/garethgeorge/backrest/commit/e6031bfa543a7300e622c1b0f56efc6320e7611e))
* support more versions of restic ([0cdfd11](https://github.com/garethgeorge/backrest/commit/0cdfd115e29a0b08d5814e71c0f4a8f2baf52e90))
* task priority not taking effect ([af7462c](https://github.com/garethgeorge/backrest/commit/af7462cefb130153cdaaa08e8ebefefa40e80e49))
* time formatting in operation list ([53c7f12](https://github.com/garethgeorge/backrest/commit/53c7f1248f5284080fff872ac79b3996474412b3))
* UI layout adjustments ([7d1b95c](https://github.com/garethgeorge/backrest/commit/7d1b95c81f0f69840ce1d20cb0d4a4bb90011dc9))

## [0.2.0](https://github.com/garethgeorge/backrest/compare/v0.1.3...v0.2.0) (2023-12-03)


### Features

* forget soft-deletes operations associated with removed snapshots ([f3dc7ff](https://github.com/garethgeorge/backrest/commit/f3dc7ffd077fef67870852f8f4e8b9aa6c94806e))
* forget soft-deletes operations associated with removed snapshots ([38bc107](https://github.com/garethgeorge/backrest/commit/38bc107db394716e34245f1edefc5e4cf4a15333))
* improve oplist performance and display forget operations in oplist ([#22](https://github.com/garethgeorge/backrest/issues/22)) ([51b4921](https://github.com/garethgeorge/backrest/commit/51b49214e3d32cc4b28e13085bd196ba164a8c19))


### Bug Fixes

* forget deadlocking and misc smaller bugs ([b7c633d](https://github.com/garethgeorge/backrest/commit/b7c633d021d68d4880a5f442ce70a858002b4af2))
* improve operation ordering to fix snapshots indexed before forget operation ([#21](https://github.com/garethgeorge/backrest/issues/21)) ([b513b08](https://github.com/garethgeorge/backrest/commit/b513b08e51434c28c90f5f062b4ae292f6854f4e))
* task priority not taking effect ([af7462c](https://github.com/garethgeorge/backrest/commit/af7462cefb130153cdaaa08e8ebefefa40e80e49))
* UI layout adjustments ([7d1b95c](https://github.com/garethgeorge/backrest/commit/7d1b95c81f0f69840ce1d20cb0d4a4bb90011dc9))

## [0.2.0](https://github.com/garethgeorge/backrest/compare/v0.1.3...v0.2.0) (2023-12-02)


### Features

* forget soft-deletes operations associated with removed snapshots ([f3dc7ff](https://github.com/garethgeorge/backrest/commit/f3dc7ffd077fef67870852f8f4e8b9aa6c94806e))
* forget soft-deletes operations associated with removed snapshots ([38bc107](https://github.com/garethgeorge/backrest/commit/38bc107db394716e34245f1edefc5e4cf4a15333))


### Bug Fixes

* forget deadlocking and misc smaller bugs ([b7c633d](https://github.com/garethgeorge/backrest/commit/b7c633d021d68d4880a5f442ce70a858002b4af2))
* improve operation ordering to fix snapshots indexed before forget operation ([#21](https://github.com/garethgeorge/backrest/issues/21)) ([b513b08](https://github.com/garethgeorge/backrest/commit/b513b08e51434c28c90f5f062b4ae292f6854f4e))
* task priority not taking effect ([af7462c](https://github.com/garethgeorge/backrest/commit/af7462cefb130153cdaaa08e8ebefefa40e80e49))
