# Changelog

## [1.10.1](https://github.com/garethgeorge/backrest/compare/v1.10.0...v1.10.1) (2025-10-31)


### Bug Fixes

* database migration fails for some users due to legacy repo_guids ([3a58090](https://github.com/garethgeorge/backrest/commit/3a58090646d0fd00c5485864d3462cf6158824c8))
* set --windows-tray=true by default ([5b3d23b](https://github.com/garethgeorge/backrest/commit/5b3d23b42276f23180728695fbc23f94a840a103))

## [1.10.0](https://github.com/garethgeorge/backrest/compare/v1.9.2...v1.10.0) (2025-10-31)


### Features

* add telegram notification hook ([#963](https://github.com/garethgeorge/backrest/issues/963)) ([b2b206e](https://github.com/garethgeorge/backrest/commit/b2b206e5351a39c9ea0323803728b0f4bcc442cc))
* allow direct downloading files and folders through backrest webui without restoring first ([#921](https://github.com/garethgeorge/backrest/issues/921)) ([d9b7715](https://github.com/garethgeorge/backrest/commit/d9b77155f3ea8b4bd44d78d3f2489e53fd4a0e6d))


### Bug Fixes

* avoid shared cache memory databases ([#926](https://github.com/garethgeorge/backrest/issues/926)) ([e8e9429](https://github.com/garethgeorge/backrest/commit/e8e94295ccc8890c048b9d8808f1ca9a5de775ce))
* getOperationEvents use create EmptySchema not javascript empty object ([#951](https://github.com/garethgeorge/backrest/issues/951)) ([ecf74f9](https://github.com/garethgeorge/backrest/commit/ecf74f9112ee51268c56f82ccb68798c37633a73))
* improve JSON parsing resilience ([#928](https://github.com/garethgeorge/backrest/issues/928)) ([e41c357](https://github.com/garethgeorge/backrest/commit/e41c357d30bd8aa16a1ee209e3fdc42f1d3dc87a))
* merge windows tray functionality into backrest.exe for single binary install on windows ([#903](https://github.com/garethgeorge/backrest/issues/903)) ([3ccb883](https://github.com/garethgeorge/backrest/commit/3ccb883b2f47880017dcb0dd3383cfdf58836929))
* remove scheduled task time limit for Windows 2012 ([#899](https://github.com/garethgeorge/backrest/issues/899)) ([6cf7df2](https://github.com/garethgeorge/backrest/commit/6cf7df23b89c4317ec75f6b3a271a6b9c19aae78))
* restic process should run in the background on windows ([20bd9a4](https://github.com/garethgeorge/backrest/commit/20bd9a49e4ed4dc64176b581f1210d80400acded))
* support backrest installation as an openrc service ([#947](https://github.com/garethgeorge/backrest/issues/947)) ([32ead81](https://github.com/garethgeorge/backrest/commit/32ead816229cf748c783fef9eed03a8407841450))
* update css to actually center elements having class `.float-center-right` ([#931](https://github.com/garethgeorge/backrest/issues/931)) ([2ced790](https://github.com/garethgeorge/backrest/commit/2ced7903402c5346c9dc6189fe083ff3fc493ef0))
* update restic version to 0.18.1 ([#910](https://github.com/garethgeorge/backrest/issues/910)) ([b34ea4c](https://github.com/garethgeorge/backrest/commit/b34ea4c5987d9c93f36e37bcf4d8cd517b4420ac))
* use default locale for formatting dates ([#950](https://github.com/garethgeorge/backrest/issues/950)) ([c4d333e](https://github.com/garethgeorge/backrest/commit/c4d333e0ed2e299be7c6e6f6692b0da01900d142))

## [1.9.2](https://github.com/garethgeorge/backrest/compare/v1.9.1...v1.9.2) (2025-08-28)


### Bug Fixes

* 'items' shows the total number of flows under the parent node rather than immediate decendents ([16e6ede](https://github.com/garethgeorge/backrest/commit/16e6edef9dc2af29430520caad503ca0a476cd78))
* auto expand parents of pending and in-progress operations in tree view ([3d97894](https://github.com/garethgeorge/backrest/commit/3d978948bb04805481d3f1fc5bfd32457924bccf))
* backrestmon should call flag.Parse to correctly handle port overrides ([da0ac09](https://github.com/garethgeorge/backrest/commit/da0ac093f19abf9ac17cc1ce310b8461ae8bf39c))
* default env variables from the docker-compose.yml sample ([c1c4d21](https://github.com/garethgeorge/backrest/commit/c1c4d218b94ed4921813cc98cf52aea4af62dcdf))
* further improve tree view layout and display preview item counts for hidden subtrees ([fe78df3](https://github.com/garethgeorge/backrest/commit/fe78df34c6d89b4a75620e46df6a53e0b6a758da))
* improve optree view layout and performance with lazy rendering ([c84a084](https://github.com/garethgeorge/backrest/commit/c84a08475f8a55d1e7c531e596653801ba2c0225))
* modernize windows installer with Inno Setup ([#867](https://github.com/garethgeorge/backrest/issues/867)) ([dc481e5](https://github.com/garethgeorge/backrest/commit/dc481e55c8b5ded17009dc2f57eaa8946467f1f0))
* reduce overly verbose logs during backups and other operations ([ca54a06](https://github.com/garethgeorge/backrest/commit/ca54a063d9e265861a70c17120f865c3a436f880))
* restic package properly handles 'verbose_status' and 'exit_error' status types ([93f7edb](https://github.com/garethgeorge/backrest/commit/93f7edbdd7b33cab6292e49df504ddf46eea4fa0))
* rework output handling in restic pkg to improve error capturing and consistency ([a271afe](https://github.com/garethgeorge/backrest/commit/a271afe7e3e840071ee3d2d18861ccd31548db89))
* use zap logger for restic log formatting ([8c3d7ef](https://github.com/garethgeorge/backrest/commit/8c3d7ef55dde1c69f662e94f56ffa5bcc7654bb2))

## [1.9.1](https://github.com/garethgeorge/backrest/compare/v1.9.0...v1.9.1) (2025-08-06)


### Bug Fixes

* new backrest installs can't set instance ID ([7f51ef5](https://github.com/garethgeorge/backrest/commit/7f51ef51e7ee3519ec22e766007638e7a884f05a))

## [1.9.0](https://github.com/garethgeorge/backrest/compare/v1.8.1...v1.9.0) (2025-08-04)


### Features

* add arg forwarding to docker-entrypoint and backrestmon ([f52275b](https://github.com/garethgeorge/backrest/commit/f52275bab3823e0f91a9cc959b1241b3e0cc4345))
* add arg forwarding to docker-entrypoint and backrestmon ([42054f4](https://github.com/garethgeorge/backrest/commit/42054f4a22f122955e74d852833a3db217561552))
* add support for block kit slack body ([#774](https://github.com/garethgeorge/backrest/issues/774)) ([9a4e0c5](https://github.com/garethgeorge/backrest/commit/9a4e0c5c74c5cdfa3ac4d78d910fcf061f560e4c))
* allow repo url to be set from env with ${RESTIC_REPOSITORY} ([#813](https://github.com/garethgeorge/backrest/issues/813)) ([39f0c09](https://github.com/garethgeorge/backrest/commit/39f0c092e48a7340e84cf1ca5dbb9f2d47deb9b0))
* multihost sync ui ([#825](https://github.com/garethgeorge/backrest/issues/825)) ([6e0c201](https://github.com/garethgeorge/backrest/commit/6e0c20102541e587494efd49e5bead07c95c3b59))
* sync api creates and uses cryptographic identity of local instance ([#780](https://github.com/garethgeorge/backrest/issues/780)) ([eb9948b](https://github.com/garethgeorge/backrest/commit/eb9948b993ea4e64b7264446faf950cbf96c2767))
* ui tweaks to improve backup / snapshot detail readability ([eaa1143](https://github.com/garethgeorge/backrest/commit/eaa1143f57077c7591f749935fd39d19f346be09))
* **webui:** improve compression graph readability ([90b77db](https://github.com/garethgeorge/backrest/commit/90b77db05a623efe9e25b5d3bd2cc46fce55e78e))


### Bug Fixes

* add docker entrypoint to set appropriate defaults for env vars ([5bc63d4](https://github.com/garethgeorge/backrest/commit/5bc63d4b9304b80315cda28dc7246c98e588d36c))
* improve resiliance to warnings printed by restic when executing JSON commands ([5112b76](https://github.com/garethgeorge/backrest/commit/5112b7676621199517312cce3116cd2c4bdcb198))
* missing KeepLastN in scheduled retention policy ([#778](https://github.com/garethgeorge/backrest/issues/778)) ([1a9bed2](https://github.com/garethgeorge/backrest/commit/1a9bed21e2b543908dffb705d4a56d0a6b64c4fa))
* **prune:** correctly handle max-unused 0% ([fe53a60](https://github.com/garethgeorge/backrest/commit/fe53a603da186a462f57f947a091059ff22cd45e))
* race condition in taskcollectgarbage potentially prematurely deletes logs for tasks currently running ([#828](https://github.com/garethgeorge/backrest/issues/828)) ([cd5814d](https://github.com/garethgeorge/backrest/commit/cd5814d84825a5cf93b1b36de446779cd9a2ac0d))
* sometimes summary dashboard doesn't load on first viewing ([f512ffe](https://github.com/garethgeorge/backrest/commit/f512ffebe3e3c79b568ab617ece9669472d49e7f))
* **webui:** clarify retention policy descriptions ([55949a2](https://github.com/garethgeorge/backrest/commit/55949a28c5e3b03d1d06e485f4a5e01d5e734b5e))
* Windows installer ToolTipIcon Info Enum ([#799](https://github.com/garethgeorge/backrest/issues/799)) ([cd67858](https://github.com/garethgeorge/backrest/commit/cd67858a3414ee42ee0f5a99995432cd7461443c))

## [1.8.1](https://github.com/garethgeorge/backrest/compare/v1.8.0...v1.8.1) (2025-05-06)


### Bug Fixes

* --keep-last n param to mitigate loss of sub-hourly snapshots ([#741](https://github.com/garethgeorge/backrest/issues/741)) ([18354c8](https://github.com/garethgeorge/backrest/commit/18354c82692e43fbb56d1d155e50cbce4c981fae))
* batch sqlite store IO to better handle large deletes in migrations ([d7c57a8](https://github.com/garethgeorge/backrest/commit/d7c57a850671f1ecc8efa11418a6fddeaf3d9d28))
* correct bug in stats panel date format for "Total Size" stats ([658514c](https://github.com/garethgeorge/backrest/commit/658514ceb8f786af63d307ff045c238eaf8eeed5))
* handle Accept-Encoding properly for compressed webui srcs ([7930b9c](https://github.com/garethgeorge/backrest/commit/7930b9c2d9bca61ccce73c5699f0ad134f005ffb))
* improve formatting of commands printed in logs for debugability ([f5c1bb9](https://github.com/garethgeorge/backrest/commit/f5c1bb90b583f644ed79ba4a27a64ad8b15fbe01))
* limit run command output to 2MB ([01d9c9f](https://github.com/garethgeorge/backrest/commit/01d9c9f3834e39c2de0a8ebc3b51e30d46afafd6))
* rare repoPool initialization crash ([59614d8](https://github.com/garethgeorge/backrest/commit/59614d84b77d0616542170fecec90303d4b973ff))
* reduce memory overhead when downloading restic updates ([715c7cc](https://github.com/garethgeorge/backrest/commit/715c7ccf130923bc001df55b322ae898219c1a00))
* update restic install script to allow newer versions of restic if present in the path ([ceeaad4](https://github.com/garethgeorge/backrest/commit/ceeaad49891fe299b3cfb47be0aebfc81b5378fa))
* use github.com/vearutop/statigz to embed webUI srcs and improve Accept-Encoding handling ([e8297b1](https://github.com/garethgeorge/backrest/commit/e8297b1ee55be5ec9823f39bc870fb5900643a1a))
* use gitlfs to track image assets for docs ([153b43b](https://github.com/garethgeorge/backrest/commit/153b43be0a178d9886b746a72187e60ac8a73674))

## [1.8.0](https://github.com/garethgeorge/backrest/compare/v1.7.3...v1.8.0) (2025-04-02)


### Bug Fixes

* deduplicate indexed snapshots ([#716](https://github.com/garethgeorge/backrest/issues/716)) ([b3b1eef](https://github.com/garethgeorge/backrest/commit/b3b1eefe9b07dbc782ad2a519f960834be1329b3))
* glob escape some linux filename characters ([#721](https://github.com/garethgeorge/backrest/issues/721)) ([190b3bf](https://github.com/garethgeorge/backrest/commit/190b3bfd0e7debf274022b64e294204a94074d1f))
* restic outputs add newline separators between log messages ([addf49c](https://github.com/garethgeorge/backrest/commit/addf49c1f37818b5e4be05db2982a0555703fa78))
* update healthchecks hook to construct urls such that query parameters are preserved ([2a24b0a](https://github.com/garethgeorge/backrest/commit/2a24b0ad5fa3583686086e58744184fe07e3e657))


### Miscellaneous Chores

* release 1.8.0 for restic version upgrade to 0.18.0 ([ad2c357](https://github.com/garethgeorge/backrest/commit/ad2c357bc3c4d54e7290b8e6a24483a8afeba1f5))

## [1.7.3](https://github.com/garethgeorge/backrest/compare/v1.7.2...v1.7.3) (2025-03-15)


### Bug Fixes

* add missing hooks for CONDITION_FORGET_{START, SUCCESS, ERROR} ([489c6f5](https://github.com/garethgeorge/backrest/commit/489c6f5b34d39d718f4ccf62ac155826685fa8d3))
* add priority fields to gotify notifications ([#678](https://github.com/garethgeorge/backrest/issues/678)) ([ec95c4a](https://github.com/garethgeorge/backrest/commit/ec95c4a8a311f63f3c033f39c8633f50d2d47be9))
* hook errors should be shown as warnings in tree view ([9f112bc](https://github.com/garethgeorge/backrest/commit/9f112bc78d7fc9609b5832b9f665dd55c9c28714))
* improve exported prometheus metrics for task execution and status ([#684](https://github.com/garethgeorge/backrest/issues/684)) ([8bafe7e](https://github.com/garethgeorge/backrest/commit/8bafe7ea35e990377f96662fc81ccdcc34b4dda6))
* index snapshots incorrectly creates duplicate entries for snapshots from other instances  ([#693](https://github.com/garethgeorge/backrest/issues/693)) ([5ab7553](https://github.com/garethgeorge/backrest/commit/5ab755393a640090b659537de900988302d3e9ea))
* occasional truncated operation history display in repo view ([3b41d9f](https://github.com/garethgeorge/backrest/commit/3b41d9fd5bd611dd0c59bcef13a3da0e2d6f02ce))
* support AWS_SHARED_CREDENTIALS_FILE for s3 authentication ([154aef4](https://github.com/garethgeorge/backrest/commit/154aef4c9a26248ec7f09c731465647b5359a995))

## [1.7.2](https://github.com/garethgeorge/backrest/compare/v1.7.1...v1.7.2) (2025-02-16)


### Bug Fixes

* convert prometheus metrics to use `gauge` type ([#640](https://github.com/garethgeorge/backrest/issues/640)) ([8c4ddee](https://github.com/garethgeorge/backrest/commit/8c4ddeea7132fb94484dc32872e00ddd3b35e44d))
* hooks fail to populate a non-nil Plan variable for system tasks ([f119e1e](https://github.com/garethgeorge/backrest/commit/f119e1e979a464e508edcb13404691ad45ac3d64))
* incorrectly formatted total size on stats panel ([#667](https://github.com/garethgeorge/backrest/issues/667)) ([d2ac114](https://github.com/garethgeorge/backrest/commit/d2ac1146ac0d2d75ca7dc51c9db076adc7170000))
* misaligned favicon ([#660](https://github.com/garethgeorge/backrest/issues/660)) ([403458f](https://github.com/garethgeorge/backrest/commit/403458f70705258906258fa77d4668d70ac176e3))
* more robust delete repo and misc repo guid related bug fixes ([146032a](https://github.com/garethgeorge/backrest/commit/146032a9d7a66c422318461b8113d6369c6cd640))
* restore path on Windows ([#631](https://github.com/garethgeorge/backrest/issues/631)) ([1a9ecc5](https://github.com/garethgeorge/backrest/commit/1a9ecc58390957523f21c6a55c809b0bf22cf978))
* snapshot still showing after forget until page is refreshed ([0600733](https://github.com/garethgeorge/backrest/commit/060073325dbbf6cc94ce6471134efb91fe191cca))

## [1.7.1](https://github.com/garethgeorge/backrest/compare/v1.7.0...v1.7.1) (2025-01-24)


### Bug Fixes

* add favicon to webui ([#649](https://github.com/garethgeorge/backrest/issues/649)) ([dd1e18c](https://github.com/garethgeorge/backrest/commit/dd1e18c9cbe6ed6ba5788ea646fc99d50e41ce25))
* local network access on macOS 15 Sequoia ([#630](https://github.com/garethgeorge/backrest/issues/630)) ([0dd360b](https://github.com/garethgeorge/backrest/commit/0dd360b4973b9f60ba706f869a1a6eb883713afd))
* only log important messages e.g. errors or summary for backup and restore commands ([82f05d8](https://github.com/garethgeorge/backrest/commit/82f05d8b809efb1a7051947cafc75ee75fd2ba5f))
* provide an option for auto-initializing repos created externally ([#650](https://github.com/garethgeorge/backrest/issues/650)) ([99264b2](https://github.com/garethgeorge/backrest/commit/99264b2469e5f04705036173a2698e6dcef25671))
* test repo configuration button ([b3cfef1](https://github.com/garethgeorge/backrest/commit/b3cfef14057540bfb0d3d2e67f66d0bbfb6c45dc))
* test repo configuration button doesn't work ([07a1561](https://github.com/garethgeorge/backrest/commit/07a1561df7aed7265cdfc9561d7bd6a2e10deac2))
* whitespace at start of path can result in invalid restore target ([47a4b52](https://github.com/garethgeorge/backrest/commit/47a4b522636370ca19d85b7ac5e1019d5b227edc))

## [1.7.0](https://github.com/garethgeorge/backrest/compare/v1.6.2...v1.7.0) (2025-01-09)


### Features

* add a "test configuration" button to aid users setting up new repos ([#582](https://github.com/garethgeorge/backrest/issues/582)) ([1bb3cd7](https://github.com/garethgeorge/backrest/commit/1bb3cd70fd8a7eb12df19eaf8f01edb075f34d48))
* change payload for healthchecks to text ([#607](https://github.com/garethgeorge/backrest/issues/607)) ([a1e3a70](https://github.com/garethgeorge/backrest/commit/a1e3a708eb583c9c7116b9842c0fcd9a04b086af))
* cont'd windows installer refinements ([#603](https://github.com/garethgeorge/backrest/issues/603)) ([b1b7fb9](https://github.com/garethgeorge/backrest/commit/b1b7fb97077150c7fd5548625c6d790a4006df08))
* improve repo view layout when backups from multiple-instances are found ([ad5d396](https://github.com/garethgeorge/backrest/commit/ad5d39643ec74a546cb6316da620e3d3bc4c8ae6))
* initial backend implementation of multihost synchronization  ([#562](https://github.com/garethgeorge/backrest/issues/562)) ([a4b4de5](https://github.com/garethgeorge/backrest/commit/a4b4de5152a0437cc2fe88b97fe808d6ef6da75d))


### Bug Fixes

* avoid ant design url rule as it requires a tld to be present ([#626](https://github.com/garethgeorge/backrest/issues/626)) ([b3402a1](https://github.com/garethgeorge/backrest/commit/b3402a18d2026a2b5998ecdae5a9802f7b3c844a))
* int overflow in exponential backoff hook error policy ([#619](https://github.com/garethgeorge/backrest/issues/619)) ([1ff69f1](https://github.com/garethgeorge/backrest/commit/1ff69f121ae4f3455e132193dffe6c4a4fa80abd))
* ogid caching for better insert / update performance ([d9cf79b](https://github.com/garethgeorge/backrest/commit/d9cf79b48a4a1846a709a1d808ade53b17389fcc))
* rare race condition in etag cache when serving webui ([dbcaa7b](https://github.com/garethgeorge/backrest/commit/dbcaa7b4fb5abe88b9e5cb2ff21f2daad81b4ee5))
* ui bugs introduced by repo guid migration ([407652c](https://github.com/garethgeorge/backrest/commit/407652c9ef8e8b00e20d76b5fa4b681a32d27a81))

## [1.6.2](https://github.com/garethgeorge/backrest/compare/v1.6.1...v1.6.2) (2024-11-26)


### Bug Fixes

* allow 'run command' tasks to proceed in parallel to other repo operations ([3397a01](https://github.com/garethgeorge/backrest/commit/3397a011a3bbbdac2f7299ea4f869cd71b2d0a22))
* allow for deleting individual operations from the list view ([aa39ead](https://github.com/garethgeorge/backrest/commit/aa39ead0e1f223e7fe7c0ce6fe4dbbc3c3050728))
* better defaults in add repo / add plan views ([4d7be23](https://github.com/garethgeorge/backrest/commit/4d7be23e8cfd959e93f202eb52c8065446446d07))
* crash on arm32 device due to bad libc dependency version for sqlite driver ([#559](https://github.com/garethgeorge/backrest/issues/559)) ([e60a4fb](https://github.com/garethgeorge/backrest/commit/e60a4fbcd7b695b03ae5402868ae3c4795cb04c6))
* garbage collection with more sensible limits grouped by plan/repo ([#555](https://github.com/garethgeorge/backrest/issues/555)) ([492beb2](https://github.com/garethgeorge/backrest/commit/492beb29352ba5e5dc824d35dfaa58eed4422b8a))
* improve memory pressure from getlogs ([592e4cf](https://github.com/garethgeorge/backrest/commit/592e4cf9dd60eaad1a660c4d69fb4ffea79c98cd))
* improve windows installer and relocate backrest on Windows to %localappdata%\programs  ([#568](https://github.com/garethgeorge/backrest/issues/568)) ([00b0c3e](https://github.com/garethgeorge/backrest/commit/00b0c3e1d256a552aa05a8a90ae05e60d35c5c96))
* make cancel button more visible for a running operation ([51a6683](https://github.com/garethgeorge/backrest/commit/51a66839ff608fa0e3e60a6a48ca5d490628368e))
* set etag header to cache webUI source ([0642f4b](https://github.com/garethgeorge/backrest/commit/0642f4b65a11daab379708d7ed813ca8d6a2140f))
* substantially improve windows installer experience ([#578](https://github.com/garethgeorge/backrest/issues/578)) ([74eb869](https://github.com/garethgeorge/backrest/commit/74eb8692638c04f49004c8312ed57123ea4b5cc2))
* tray app infers UI port from BACKREST_PORT or --bind-address if available ([c810d27](https://github.com/garethgeorge/backrest/commit/c810d27375c39a9938ad4bde433dfe5997d56bfa))
* update resticinstaller to use the same binary name across versions and to use system restic install when possible ([5fea5fd](https://github.com/garethgeorge/backrest/commit/5fea5fdefdc2bad7fccb1f0cc0ea57fbe79bbcbb))
* use command mode when executing powershell scripts on windows ([#569](https://github.com/garethgeorge/backrest/issues/569)) ([57f9aeb](https://github.com/garethgeorge/backrest/commit/57f9aeb72a6db240824998cff91c0921c68a336a))
* webui may duplicate elements in a multi-instance repo ([bf77bab](https://github.com/garethgeorge/backrest/commit/bf77baba06c7296ade830e10238f1a02d0cea95c))

## [1.6.1](https://github.com/garethgeorge/backrest/compare/v1.6.0...v1.6.1) (2024-10-20)


### Bug Fixes

* login form has no background ([4fc28d6](https://github.com/garethgeorge/backrest/commit/4fc28d68a60721d333be96df2030ce53b04fbf55))
* stats operation occasionally runs twice in a row ([36543c6](https://github.com/garethgeorge/backrest/commit/36543c681ac1f138e4d207f96c143b1d1ffd84fe))
* tarlog migration fails on new installs ([5617f3f](https://github.com/garethgeorge/backrest/commit/5617f3fbe2aa5278c2b8b1903997980a9e2e16b0))

## [1.6.0](https://github.com/garethgeorge/backrest/compare/v1.5.1...v1.6.0) (2024-10-20)


### Features

* add a summary dashboard as the "main view" when backrest opens ([#518](https://github.com/garethgeorge/backrest/issues/518)) ([4b3c7e5](https://github.com/garethgeorge/backrest/commit/4b3c7e53d5b8110c179c486c3423ef9ff72feb8f))
* add watchdog thread to reschedule tasks when system time changes ([66a5241](https://github.com/garethgeorge/backrest/commit/66a5241de8cf410d0766d7e70de9b8f87e6aaddd))
* initial support for healthchecks.io notifications ([#480](https://github.com/garethgeorge/backrest/issues/480)) ([f6ee51f](https://github.com/garethgeorge/backrest/commit/f6ee51fce509808d8dde3d2af21d10994db381ca))
* migrate oplog history from bbolt to sqlite store ([#515](https://github.com/garethgeorge/backrest/issues/515)) ([0806eb9](https://github.com/garethgeorge/backrest/commit/0806eb95a044fd5f1da44aff7713b0ca21f7aee5))
* support --skip-if-unchanged ([afcecae](https://github.com/garethgeorge/backrest/commit/afcecaeb3064782788a4ff41fc31a541d93e844f))
* track long running generic commands in the oplog ([#516](https://github.com/garethgeorge/backrest/issues/516)) ([28c3172](https://github.com/garethgeorge/backrest/commit/28c31720f249763e2baee43671475c128d17b020))
* use react-router to enable linking to webUI pages ([#522](https://github.com/garethgeorge/backrest/issues/522)) ([fff3dbd](https://github.com/garethgeorge/backrest/commit/fff3dbd299163b916ae0c6819c9c0170e2e77dd9))
* use sqlite logstore ([#514](https://github.com/garethgeorge/backrest/issues/514)) ([4d557a1](https://github.com/garethgeorge/backrest/commit/4d557a1146b064ee41d74c80667adcd78ed4240c))


### Bug Fixes

* expand env vars in flags i.e. of the form ${MY_ENV_VAR} ([d7704cf](https://github.com/garethgeorge/backrest/commit/d7704cf057989af4ed2f03e81e46a6a924f833cd))
* gorelaeser docker image builds for armv6 and armv7 ([4fa30e3](https://github.com/garethgeorge/backrest/commit/4fa30e3f7ee7456d2bdf4afccb47918d01bdd32e))
* plan/repo settings button hard to click ([ec89cfd](https://github.com/garethgeorge/backrest/commit/ec89cfde518e3c38697e6421fa7e1bca31040602))

## [1.5.1](https://github.com/garethgeorge/backrest/compare/v1.5.0...v1.5.1) (2024-09-18)


### Bug Fixes

* **docs:** correct minor spelling and grammar errors ([#479](https://github.com/garethgeorge/backrest/issues/479)) ([df55681](https://github.com/garethgeorge/backrest/commit/df5568132b56d38f0ce155e546ff110a943ad87a))
* prunepolicy.max_unused_percent should allow decimal values ([3056203](https://github.com/garethgeorge/backrest/commit/3056203127b4ced26e69da2a7540d4b139dcd8e9))
* stats panel can fail to load when an incomplete operation is in the log ([d59c6fc](https://github.com/garethgeorge/backrest/commit/d59c6fc1bed06718c49fc87bfc5bf143a10ac5ed))
* update to newest restic bugfix release 0.17.1 ([d2650fd](https://github.com/garethgeorge/backrest/commit/d2650fdd591f2bdb08dce8fe55afaba0a5659e31))
* windows installation for restic 0.17.1 ([#474](https://github.com/garethgeorge/backrest/issues/474)) ([4da9d89](https://github.com/garethgeorge/backrest/commit/4da9d89749fd1bdfd9701c8efb83b69a7eef3395))

## [1.5.0](https://github.com/garethgeorge/backrest/compare/v1.4.0...v1.5.0) (2024-09-10)


### Features

* add prometheus metrics ([#459](https://github.com/garethgeorge/backrest/issues/459)) ([daacf28](https://github.com/garethgeorge/backrest/commit/daacf28699c18b27256cb4bf2eb3d9caf94a5ce8))
* compact the scheduling UI and use an enum for clock configuration ([#452](https://github.com/garethgeorge/backrest/issues/452)) ([9205da1](https://github.com/garethgeorge/backrest/commit/9205da1d2380410d1ccc4507008f28d4fa60dd32))
* implement 'on error retry' policy ([#428](https://github.com/garethgeorge/backrest/issues/428)) ([038bc87](https://github.com/garethgeorge/backrest/commit/038bc87070361ff3b7d9a90c075787e9ff3948f7))
* implement scheduling relative to last task execution ([#439](https://github.com/garethgeorge/backrest/issues/439)) ([6ed1280](https://github.com/garethgeorge/backrest/commit/6ed1280869bf42d1901ca09a5cc6b316a1cd8394))
* support live logrefs for in-progress operations ([#456](https://github.com/garethgeorge/backrest/issues/456)) ([bfaad8b](https://github.com/garethgeorge/backrest/commit/bfaad8b69e95e13006d3f64e6daa956dc060833c))


### Bug Fixes

* apply oplog migrations correctly using new storage interface ([491a6a6](https://github.com/garethgeorge/backrest/commit/491a6a67254e40167b6937f6844123de704d5182))
* backrest can erroneously show 'forget snapshot' button for restore entries ([bfde425](https://github.com/garethgeorge/backrest/commit/bfde425c2d03b0e4dc7c19381cb604dcba9d36e3))
* broken refresh and sizing for mobile view in operation tree ([0d01c5c](https://github.com/garethgeorge/backrest/commit/0d01c5c31773de996465574e77bc90fa64586e59))
* bugs in displaying repo / plan / activity status ([cceda4f](https://github.com/garethgeorge/backrest/commit/cceda4fdea5f6c2072e8641d33fffe160613dcf7))
* double display of snapshot ID for 'Snapshots' in operation tree ([80dbe91](https://github.com/garethgeorge/backrest/commit/80dbe91729efebe88d4ad8e9c4160d48254d0fc1))
* hide system operations in tree view ([8c1cf79](https://github.com/garethgeorge/backrest/commit/8c1cf791bbc2a5fc0ff279f9ba52d372c123f2d2))
* misc bugs in restore operation view and activity bar view ([656ac9e](https://github.com/garethgeorge/backrest/commit/656ac9e1b2f2ce82f5afd4a20a729b710d19c541))
* misc bugs related to new logref support ([97e3f03](https://github.com/garethgeorge/backrest/commit/97e3f03b78d9af644aaa9f4b2e4882514c85025a))
* misc logging improvements ([1879ddf](https://github.com/garethgeorge/backrest/commit/1879ddfa7991f44bd54d3de9d14d7b7c03472c78))
* new config validations make it harder to lock yourself out of backrest ([c419861](https://github.com/garethgeorge/backrest/commit/c4198619aa93fa216b9b2744cb7e4214e23c6ac6))
* reformat tags row in operation list ([0eb560d](https://github.com/garethgeorge/backrest/commit/0eb560ddfb46f33d8404d0e7ac200d7574f64797))
* remove migrations for fields that have been since backrest 1.0.0 ([#453](https://github.com/garethgeorge/backrest/issues/453)) ([546482f](https://github.com/garethgeorge/backrest/commit/546482f11533668b58d5f5eead581a053b19c28d))
* restic cli commands through 'run command' are cancelled when closing dialogue ([bb00afa](https://github.com/garethgeorge/backrest/commit/bb00afa899b17c23f6375a5ee23d3c5354f5df4d))
* simplify auth handling ([6894128](https://github.com/garethgeorge/backrest/commit/6894128d90c1d50c9da53276e4dd6b37c5357402))
* test fixes for windows file restore ([44585ed](https://github.com/garethgeorge/backrest/commit/44585ede613b87189c38f5cd456a109e653cdf64))
* UI quality of life improvements ([cc173aa](https://github.com/garethgeorge/backrest/commit/cc173aa7b1b9dda10cfb14ca179c9701d15f22f5))
* use 'restic restore &lt;snapshot id&gt;:<path>' for restore operations ([af09e47](https://github.com/garethgeorge/backrest/commit/af09e47cdda921eb11cab970939740adb1612af4))
* write debug-level logs to data dir on all platforms ([a9eb786](https://github.com/garethgeorge/backrest/commit/a9eb786db90f977984b13c3bda7f764d6dadbbef))

## [1.4.0](https://github.com/garethgeorge/backrest/compare/v1.3.1...v1.4.0) (2024-08-15)


### Features

* accept up to 2 decimals of precision for check % and prune % policies ([5374273](https://github.com/garethgeorge/backrest/commit/53742736f9dec217527ad50caed9a488da39ad45))
* add UI support for new summary details introduced in restic 0.17.0 ([4859e52](https://github.com/garethgeorge/backrest/commit/4859e528c73853d4597c5ef54d3054406a5c7e44))
* start tracking snapshot summary fields introduced in restic 0.17.0 ([505765d](https://github.com/garethgeorge/backrest/commit/505765dff978c5ecabe1986907b4c4c0c5112daf))
* update to restic 0.17.0 ([#416](https://github.com/garethgeorge/backrest/issues/416)) ([500f2ee](https://github.com/garethgeorge/backrest/commit/500f2ee6c0d8bcf65a37462d3d03452cd9dff817))


### Bug Fixes

* activitybar does not reset correctly when an in-progress operation is deleted ([244fe7e](https://github.com/garethgeorge/backrest/commit/244fe7edd203b566709dc7f14091865bc9ed6700))
* add condition_snapshot_success to .EventName ([#410](https://github.com/garethgeorge/backrest/issues/410)) ([c45f0f3](https://github.com/garethgeorge/backrest/commit/c45f0f3c668df44ba82e0d6faf73cfd8f39f0c2a))
* backrest should only initialize repos explicitly added through WebUI ([62a97a3](https://github.com/garethgeorge/backrest/commit/62a97a335df3858a53eba34e7b7c0f69e3875d88))
* forget snapshot by ID should not require a plan ([49e46b0](https://github.com/garethgeorge/backrest/commit/49e46b04a06eb75829df2f97726d850749e29b74))
* hide cron options for hours/minutes/days of week for infrequent schedules ([7c091e0](https://github.com/garethgeorge/backrest/commit/7c091e05973addaa35850774320f5e49fe016437))
* improve debug output when trying to configure a new repo ([11b3e99](https://github.com/garethgeorge/backrest/commit/11b3e9915211c8c4a06f9f6f0c30f07f005a0036))
* possible race condition leading to rare panic in GetOperationEvents ([f250adf](https://github.com/garethgeorge/backrest/commit/f250adf4a025dcb64cb569a8cb26fa0443b56fae))
* run list snapshots after updating repo config or adding new repo ([48626b9](https://github.com/garethgeorge/backrest/commit/48626b923ea5022d9c4f2075d5c2c1ec19089499))
* use addrepo RPC to apply validations when updating repo config ([a67c29b](https://github.com/garethgeorge/backrest/commit/a67c29b57ac7154bda87a7a460af26adacf6d11b))

## [1.3.1](https://github.com/garethgeorge/backrest/compare/v1.3.0...v1.3.1) (2024-07-12)


### Bug Fixes

* add docker-cli to alpine backrest image ([b6f9129](https://github.com/garethgeorge/backrest/commit/b6f9129d3042a3785106ecd24801a55b80b38146))
* add major and major.minor semantic versioned docker releases ([8db2578](https://github.com/garethgeorge/backrest/commit/8db2578b95d50dcd4abaac851c8a1a5b6e9bf15c))
* plan _system_ not found bug when running health operations ([c19665a](https://github.com/garethgeorge/backrest/commit/c19665ab063a32e2cb0ca73a4e0eaa4cee793601))

## [1.3.0](https://github.com/garethgeorge/backrest/compare/v1.2.1...v1.3.0) (2024-07-11)


### Features

* improve hook UX and execution model ([#357](https://github.com/garethgeorge/backrest/issues/357)) ([4d0d13e](https://github.com/garethgeorge/backrest/commit/4d0d13e39802fcf18186723372608d96b9bd58b0))


### Bug Fixes

* cannot run path relative executable errors on Windows ([c3ec9ee](https://github.com/garethgeorge/backrest/commit/c3ec9eeb4b5aa37e66ad115528b6708d438e9459))
* improve handling of restore operations ([620caed](https://github.com/garethgeorge/backrest/commit/620caed7e3570aa7f7cb5f7279c8b6bb277d95fc))
* operation tree key conflicts ([2dc5595](https://github.com/garethgeorge/backrest/commit/2dc55951d7047e395c0b770bc8e4d1a80ffd32d7))

## [1.2.1](https://github.com/garethgeorge/backrest/compare/v1.2.0...v1.2.1) (2024-07-02)


### Bug Fixes

* AddPlanModal and AddRepoModal should only be closeable explicitly ([15f92fc](https://github.com/garethgeorge/backrest/commit/15f92fcd901da8c06ebd94576b09879e68bf5bc5))
* disable sorting for excludes and iexcludes ([d7425b5](https://github.com/garethgeorge/backrest/commit/d7425b589376595999d3e3f401bb7ef77ffde991))
* github actions release flow for windows installers ([90e0656](https://github.com/garethgeorge/backrest/commit/90e0656fc41a2b90ee24d598023ccc6996a64b9c))
* make instance ID required field ([7c8ded2](https://github.com/garethgeorge/backrest/commit/7c8ded2fcc4b597e21c24f451e02cc14ba9a015c))
* operation tree UI bugs ([76ce3c1](https://github.com/garethgeorge/backrest/commit/76ce3c177b6a92c105c874e459bd57e1122b5ce8))
* restore always uses ~/Downloads path ([955771e](https://github.com/garethgeorge/backrest/commit/955771e1cc6bb7b143ef5c51ef9e1e09509f76b1))

## [1.1.0](https://github.com/garethgeorge/backrest/compare/v1.0.0...v1.1.0) (2024-06-01)


### Features

* add windows installer and tray app ([#294](https://github.com/garethgeorge/backrest/issues/294)) ([8a7543c](https://github.com/garethgeorge/backrest/commit/8a7543c7bf7f245d87fa079c477c50b333dfba37))
* support nice/ionice as a repo setting ([#309](https://github.com/garethgeorge/backrest/issues/309)) ([0c9f366](https://github.com/garethgeorge/backrest/commit/0c9f366e439b57007259a2ca305ac00733638566))
* support restic check operation ([#303](https://github.com/garethgeorge/backrest/issues/303)) ([ce42f68](https://github.com/garethgeorge/backrest/commit/ce42f68d0d563defabbaafce63313fcf3835d2dd))


### Bug Fixes

* collection of ui refresh timing bugs ([b218bc9](https://github.com/garethgeorge/backrest/commit/b218bc9409bf4a6c70da06e1f98760ff520afc37))
* improve prune and check scheduling in new repos ([c58055e](https://github.com/garethgeorge/backrest/commit/c58055ec91ccc9a8afc5d3ff402f68da00a04e66))
* release workflow ([290d018](https://github.com/garethgeorge/backrest/commit/290d018c7585a4032b5f5d7a26f06e4d74f8b5cb))
* snapshot browser on Windows ([19ed611](https://github.com/garethgeorge/backrest/commit/19ed611477186af2702fb7ba403b0bac45ccc4aa))
* UI refresh timing bugs ([ba005ae](https://github.com/garethgeorge/backrest/commit/ba005aee0beb0105948901330e9ab7f7290eec92))

## [1.0.0](https://github.com/garethgeorge/backrest/compare/v0.17.2...v1.0.0) (2024-05-20)


### ⚠ BREAKING CHANGES

* redefine hostname as a required property that maps to --host ([#256](https://github.com/garethgeorge/backrest/issues/256))

### Features

* add CONDITION_SNAPSHOT_WARNING hook triggered by any warning status at the completion of a snapshot ([f0ee20f](https://github.com/garethgeorge/backrest/commit/f0ee20f53de58e0a0a0a63137e203161d8acce4d))
* add download link to create a zip archive of restored files ([a75a5c2](https://github.com/garethgeorge/backrest/commit/a75a5c2297df4eb89235a54efd38d9539b7c15e5))
* add force kill signal handler that dumps stacks ([386f46a](https://github.com/garethgeorge/backrest/commit/386f46a090e6df28f28cbca15d992ce4ad6d5dd5))
* add seek support to join iterator for better performance ([802146a](https://github.com/garethgeorge/backrest/commit/802146a6c023779d6e5e0879994ec7dc5479e304))
* ensure instance ID is set for all operations ([65d4a1d](https://github.com/garethgeorge/backrest/commit/65d4a1df0e9e717f5f88d7c5bec37f18d877b876))
* implement 'run command' button to execute arbitrary restic commands in a repo ([fbad981](https://github.com/garethgeorge/backrest/commit/fbad981a1d3ae75c1eeebf9fd3bf4cef4f72b4c4))
* improve support for instance ID tag ([be0cdd5](https://github.com/garethgeorge/backrest/commit/be0cdd59be270e0393dc4d587bfa708c610ac0a5))
* keep a rolling backup of the last 10 config versions ([1a053f2](https://github.com/garethgeorge/backrest/commit/1a053f274846e822ecfd3c76e0d1b4860fada58a))
* overhaul task interface and introduce 'flow ID' for simpler grouping of operations ([#253](https://github.com/garethgeorge/backrest/issues/253)) ([7a10bdc](https://github.com/garethgeorge/backrest/commit/7a10bdca7b00f337a2c85780861e479b7aa35cb5))
* redefine hostname as a required property that maps to --host ([#256](https://github.com/garethgeorge/backrest/issues/256)) ([4847010](https://github.com/garethgeorge/backrest/commit/484701007ff2f7f80fff308827b1af456a78cbb9))
* support env variable substitution e.g. FOO=${MY_FOO_VAR} ([8448f4c](https://github.com/garethgeorge/backrest/commit/8448f4cc3aebd1b481fc695c2aa0d02e18689a20))
* unified scheduling model ([#282](https://github.com/garethgeorge/backrest/issues/282)) ([531cd28](https://github.com/garethgeorge/backrest/commit/531cd286d87c8004b95bfd9b4512dffccc6d500d))
* update snapshot management to track and filter on instance ID, migrate existing snapshots ([5a996d7](https://github.com/garethgeorge/backrest/commit/5a996d74b06dcf6c1439cac9134ec51ba7167c15))
* validate plan ID and repo ID ([f314c7c](https://github.com/garethgeorge/backrest/commit/f314c7cced2db23a4008622c97a27697c832c664))


### Bug Fixes

* add virtual root node to snapshot browser ([6045c87](https://github.com/garethgeorge/backrest/commit/6045c87cdf5a68afd81203602ee5827eda5af8e7))
* additional tooltips for add plan modal ([fcdf07d](https://github.com/garethgeorge/backrest/commit/fcdf07da6c330aed7fea017835cbbf56679b3749))
* adjust task priorities ([756e64a](https://github.com/garethgeorge/backrest/commit/756e64a2002aead213d67c8d37d851688af51168))
* center-right align settings icons for plans/repos ([982e2fb](https://github.com/garethgeorge/backrest/commit/982e2fb2cd84fe193a4b37bda8c21f75c8eb3382))
* concurrency issues in run command handler ([411a4fb](https://github.com/garethgeorge/backrest/commit/411a4fb6f00fd46f1fbdb0b8e3a971d016a6e0f8))
* date formatting ([b341146](https://github.com/garethgeorge/backrest/commit/b341146fce40ee8bdaf771c4c5269160198b6386))
* downgrade omission of 'instance' field from an error to a warning ([6ae82f7](https://github.com/garethgeorge/backrest/commit/6ae82f70d456c05b3ad0ab01e901be8bd01bb9eb))
* error formatting for repo init ([1a3ace9](https://github.com/garethgeorge/backrest/commit/1a3ace90141a48e949c6c796fa8445de134baa98))
* hide successful hook executions in the backup view ([65bb8ef](https://github.com/garethgeorge/backrest/commit/65bb8ef14b77cfe07c2db26e0fcc8e0bbc1a9287))
* improve cmd error formatting now that logs are available for all operations ([6eb704f](https://github.com/garethgeorge/backrest/commit/6eb704f07bfae1cfc25208bc1a20908d229f344e))
* improve concurrency handling in RunCommand ([07b0950](https://github.com/garethgeorge/backrest/commit/07b09502b9554386afa7bd4c5487f9b8da3a59bb))
* improve download speeds for restored files ([eb07931](https://github.com/garethgeorge/backrest/commit/eb079317c05946fb74a74e59592940ada9eef4ea))
* install.sh was calling systemctl on Darwin ([#260](https://github.com/garethgeorge/backrest/issues/260)) ([f6d5837](https://github.com/garethgeorge/backrest/commit/f6d58376b76707de36d851808812d6b3384e2ca9))
* minor bugs and tweak log rotation history to 14 days ([ad9a770](https://github.com/garethgeorge/backrest/commit/ad9a77029ce07a5bb7da2738b108d0f93cb57440))
* miscellaneous bug fixes ([df4be0f](https://github.com/garethgeorge/backrest/commit/df4be0f7bc014a3862f14fcf79cffc53f45c6ea0))
* prompt for user action to set an instance ID on upgrade ([294864f](https://github.com/garethgeorge/backrest/commit/294864fe433302571ba9ff9eb7c2dd475fa1c560))
* rebase stats panel onto a better chart library ([b22028e](https://github.com/garethgeorge/backrest/commit/b22028eb4f185be96ff4407fccafa2d1cdf491a1))
* reserve IDs starting and ending with '__' for internal use ([711064f](https://github.com/garethgeorge/backrest/commit/711064fb0017830bc148643617ca8da5aa0add41))
* retention policy display may show default values for some fields ([9d6c1ba](https://github.com/garethgeorge/backrest/commit/9d6c1baf87c31b7a2cfb633fdd228d58021f7b0f))
* run stats after every prune operation ([7fce593](https://github.com/garethgeorge/backrest/commit/7fce59311d531cb9058965cde780f8930cd98a9b))
* schedule view bug ([0764804](https://github.com/garethgeorge/backrest/commit/0764804ea558df6edd5e65ca1ea9c843a75fc147))
* secure download URLs when downloading tar archive of exported files ([a30d5ef](https://github.com/garethgeorge/backrest/commit/a30d5efe1c354dd6f6c91d3b1465a244077e1e47))
* UI fixes for restore row and settings modal ([e9d6cbe](https://github.com/garethgeorge/backrest/commit/e9d6cbeaff03675928e036461a999cb4bde64e54))
* use int64 for large values in structs for compatibility with 32bit devices ([#250](https://github.com/garethgeorge/backrest/issues/250)) ([84b4b68](https://github.com/garethgeorge/backrest/commit/84b4b68760ded53d9bda2fbc992646f309094f52))
* use locale to properly format time ([89a49c1](https://github.com/garethgeorge/backrest/commit/89a49c1fa7c6cafedef30bdf695e76920e2c690c))

## [0.17.2](https://github.com/garethgeorge/backrest/compare/v0.17.1...v0.17.2) (2024-04-18)

### Bug Fixes

- add tini to docker images to reap rclone processes left behind by restic ([6408518](https://github.com/garethgeorge/backrest/commit/6408518582fb2a1b529f5c9fb0c595df230f3df6))
- armv7 support for docker releases ([ec39533](https://github.com/garethgeorge/backrest/commit/ec39533e4cddf2f0354ec3fcb4c52ba37a9b00ec))
- bug in new task queue implementation ([5d6074e](https://github.com/garethgeorge/backrest/commit/5d6074eb296e6737f1959fba913c67e09e60ef47))
- improve restic pkg's output handling and buffering ([aacdf9b](https://github.com/garethgeorge/backrest/commit/aacdf9b7cd529a6f677cd7f1d9ed2fbbcadc9b8a))
- Linux ./install.sh script fails when used for updating backrest ([#226](https://github.com/garethgeorge/backrest/issues/226)) ([be09303](https://github.com/garethgeorge/backrest/commit/be0930368b83ba8f159b28bc286300c56bd6a3a3))
- use new orchestrator queue ([4a81889](https://github.com/garethgeorge/backrest/commit/4a81889d810d409ed42fcf07a0fa6a4ac97db72b))

## [0.17.1](https://github.com/garethgeorge/backrest/compare/v0.17.0...v0.17.1) (2024-04-12)

### Bug Fixes

- revert orchestrator changes ([07cffcb](https://github.com/garethgeorge/backrest/commit/07cffcb5d8dc018631fcd0d1f98cc01553a6574e))

## [0.17.0](https://github.com/garethgeorge/backrest/compare/v0.16.0...v0.17.0) (2024-04-12)

### Features

- add a Bash script to help Linux user manage Backrest ([#187](https://github.com/garethgeorge/backrest/issues/187)) ([d78bcfa](https://github.com/garethgeorge/backrest/commit/d78bcfab845a86523868a91fe200b2a3c4c07e07))
- allow hook exit codes to control backup execution (e.g fail, skip, etc) ([c4ae5b3](https://github.com/garethgeorge/backrest/commit/c4ae5b3f2257d6c04ed08188cfc509023137b460))
- release backrest as a homebrew tap ([16a7d0e](https://github.com/garethgeorge/backrest/commit/16a7d0e95ae51c9f86e2d38e2c494b324245a9db))
- use amd64 restic for arm64 Windows ([#201](https://github.com/garethgeorge/backrest/issues/201)) ([3770966](https://github.com/garethgeorge/backrest/commit/3770966111f096c84b4702e6639397e8efab93a7))
- use new task queue implementation in orchestrator ([1d04898](https://github.com/garethgeorge/backrest/commit/1d0489847e6fee5baed807117379738aceca4a2d))

### Bug Fixes

- address minor data race in command output handling and enable --race in coverage ([3223138](https://github.com/garethgeorge/backrest/commit/32231385ed20c0dccda12361eaac7cc088ec15a0))
- bugs in refactored task queue and improved coverage ([834b74f](https://github.com/garethgeorge/backrest/commit/834b74f0f3eec42055d1af6ecfe34d448f71d97b))
- cannot set retention policy buckets to 0 ([7e9bf15](https://github.com/garethgeorge/backrest/commit/7e9bf15976006c7f3ff96948d2b2c291737c9e88))
- **css:** fixing overflow issue ([#191](https://github.com/garethgeorge/backrest/issues/191)) ([1d9e43e](https://github.com/garethgeorge/backrest/commit/1d9e43e49b21adc4ed8ce1ec96199084981d709a))
- default BACKREST_PORT to 127.0.0.1:9898 (localhost only) when using install.sh ([eb07230](https://github.com/garethgeorge/backrest/commit/eb07230cc0843643406fa44ca21c3a138baced77))
- handle backpressure correctly in event stream ([4e2bf1f](https://github.com/garethgeorge/backrest/commit/4e2bf1f76c4d35d61fc48111baaa33b7b7a8c133))
- improve tooltips on AddRepoModal ([e2be189](https://github.com/garethgeorge/backrest/commit/e2be189f9e4bb617a69e4b9c15da3d1920549349))
- include ioutil helpers ([88a926b](https://github.com/garethgeorge/backrest/commit/88a926b0a3a52efb82da5df3423a001ed140639c))
- limit cmd log length to 32KB per operation ([92d52be](https://github.com/garethgeorge/backrest/commit/92d52bed8e84d6cd8dd331a1fa52a6e2d30cb7a7))
- misc UI and backend bug fixes ([e96f403](https://github.com/garethgeorge/backrest/commit/e96f4036df6849650d6e378c9a175fef86b2962b))
- refactor priority ordered task queue implementation ([8b9280e](https://github.com/garethgeorge/backrest/commit/8b9280ed57b84b7da814e285542c57b7c14209ae))
- spawn goroutine to update oplog with progress during backup/restore ([eab1c1b](https://github.com/garethgeorge/backrest/commit/eab1c1bffe2a1aec6afa9e054278ff98ca3047cf))
- use C:\Program Files\backrest on both x64 and 32-bit ([#200](https://github.com/garethgeorge/backrest/issues/200)) ([7b0d3aa](https://github.com/garethgeorge/backrest/commit/7b0d3aa1be7bc93363b00154d09502b4e4e63ba5))

## [0.16.0](https://github.com/garethgeorge/backrest/compare/v0.15.1...v0.16.0) (2024-03-30)

### Features

- allow disabling authentication ([8429174](https://github.com/garethgeorge/backrest/commit/84291746af5fc863f90bcf7ae9ba5a2d3ca26cdd))
- improve consistency of restic command execution and output capture ([16e22aa](https://github.com/garethgeorge/backrest/commit/16e22aa623c5a0a6e6b0e6df12a8e3d09c2ff31f))
- improve observability by exposing restic command logs in UI ([eeb8c8e](https://github.com/garethgeorge/backrest/commit/eeb8c8e6b377f96c0c39bd2b169b86986933d570))
- make hostname configurable in settings panel ([2e4e3cf](https://github.com/garethgeorge/backrest/commit/2e4e3cf9c78cac587a3a40635ec068726b3f4d2d))
- sort lists in configuration ([6f330ac](https://github.com/garethgeorge/backrest/commit/6f330ac37b8ce621fbe82594c41d6f5091f03dfd))
- support shoutrrr notification service ([fa6407c](https://github.com/garethgeorge/backrest/commit/fa6407cac25ed8f0a32cc9ed5fdd8454bc9abbe5))
- switch alpine as the default base image for docker releases ([7425c9b](https://github.com/garethgeorge/backrest/commit/7425c9bb0e08cf650e596ae43a736507313e3f2f))
- update macos install script to set PATH env var for use with rclone ([8cf43f2](https://github.com/garethgeorge/backrest/commit/8cf43f28921ef7182f1c655fa82470e74698d3ce))

### Bug Fixes

- add new logs to orchestrator and increase clock change polling to every 5 minutes ([5b7e2b0](https://github.com/garethgeorge/backrest/commit/5b7e2b080d31a2f77a5f9b6737dfbb84cfb63cce))
- api path relative to UI serving location to support reverse proxies with prefix stripping ([ac7f24e](https://github.com/garethgeorge/backrest/commit/ac7f24ed04679ed6cc3ea779325c0e0b49c9f526))
- cleanup spacing and hook titles in AddRepoModal and AddPlanModal ([c32874c](https://github.com/garethgeorge/backrest/commit/c32874c1d6fc8292a2fb91f0b22c7146083bc468))
- correctly auto-expand first 5 backups when opening plan/repo ([d7ca35b](https://github.com/garethgeorge/backrest/commit/d7ca35b66f61c12360905e98b775e3256210176e))
- include error messages in restic logs ([b68f7c6](https://github.com/garethgeorge/backrest/commit/b68f7c69138d516f84f9fca3040003604bff24e6))
- include restic binary in alpine and scratch docker images ([f7bd9f7](https://github.com/garethgeorge/backrest/commit/f7bd9f7d0a9c62baedd1a341eb76e836fb00cfa5))
- incorrectly indicate AM/PM in formatted date strings ([5d34e0b](https://github.com/garethgeorge/backrest/commit/5d34e0bfb5cffd44d971b0e1052574fe640049e7))
- make notification title optional on discord notifications ([e8bbe2c](https://github.com/garethgeorge/backrest/commit/e8bbe2c8f509de67181750f8451fae841b3fa195))
- make tree view the default panel for repo overview ([3f9c9f4](https://github.com/garethgeorge/backrest/commit/3f9c9f4ff8bea0f79b03222609d7c302e241bab2))
- tasks duplicated when config is updated during a running operation ([035684c](https://github.com/garethgeorge/backrest/commit/035684ca343b47dfb3f131c89e15f06e8155f550))

## [0.15.1](https://github.com/garethgeorge/backrest/compare/v0.15.0...v0.15.1) (2024-03-19)

### Bug Fixes

- forget operations failing with new retention policy format ([0a059bb](https://github.com/garethgeorge/backrest/commit/0a059bbb39ea6d5f6f989cc4a4541ec8aedbc071))

## [0.15.0](https://github.com/garethgeorge/backrest/compare/v0.13.0...v0.15.0) (2024-03-19)

### Features

- add 'compute stats' button to refresh stats on repo view ([1f42b6a](https://github.com/garethgeorge/backrest/commit/1f42b6ab4e0313bbb12e6bc22b561d7544504644))
- add option to disable scheduled execution of a plan ([aea74c5](https://github.com/garethgeorge/backrest/commit/aea74c51c0fb3908ece57f813c9ae6190e1fd46b))
- add release artifacts for arm32 ([a737371](https://github.com/garethgeorge/backrest/commit/a737371ed559f5b65e734b0d97c44dcb2749ce53))
- automatically remove Apples quarantine flag ([#155](https://github.com/garethgeorge/backrest/issues/155)) ([3e76beb](https://github.com/garethgeorge/backrest/commit/3e76bebd054eb7bfc9f8da4681459b863ae50c55))
- check for basic auth ([#110](https://github.com/garethgeorge/backrest/issues/110)) ([#129](https://github.com/garethgeorge/backrest/issues/129)) ([871c54f](https://github.com/garethgeorge/backrest/commit/871c54f35f8651632714ca7d3a3ab0e809549b51))
- improved stats visualization with graphs and cleanup operation filtering ([5b362cc](https://github.com/garethgeorge/backrest/commit/5b362ccbb45e59954dad574b93848195d45b55ef))
- pass through all env variables from parent process to restic ([24afd51](https://github.com/garethgeorge/backrest/commit/24afd514ad80f542e6e1862d1c42195c6fbe1b47))
- support flag overrides for 'restic backup' in plan configuration ([56f5e40](https://github.com/garethgeorge/backrest/commit/56f5e405037a6309a3d1299356b363cd84281aef))
- use disambiguated retention policy format ([5a5a229](https://github.com/garethgeorge/backrest/commit/5a5a229f456bf3d4d34cb4751c2a2ff3b6907511))

### Bug Fixes

- alpine linux Dockerfile and add openssh ([3cb9d27](https://github.com/garethgeorge/backrest/commit/3cb9d2717c1bda7bb7ed4e029ac938c851b9f664))
- backrest shows hidden operations in list view ([c013f06](https://github.com/garethgeorge/backrest/commit/c013f069ff5eab6177d2bde373f23efe34b1aa8d))
- BackupInfoCollector handling of filtered events ([f1e4619](https://github.com/garethgeorge/backrest/commit/f1e4619e9d98416289fb0ee51d56ff48e163b85d))
- bugs in env var validation and form field handling ([7e909c4](https://github.com/garethgeorge/backrest/commit/7e909c4a96b053e8093f3b4f3d26c46b1c618947))
- compression progress ratio should be float64 ([1759b5d](https://github.com/garethgeorge/backrest/commit/1759b5dc55ab17a1c76d47adee7f4e21f7ef09f5))
- handle timezone correctly with tzdata package on alpine ([0e94f30](https://github.com/garethgeorge/backrest/commit/0e94f308cde40059f9c4104ed21f8c701a349c57))
- install rclone with apk for alpine image ([#138](https://github.com/garethgeorge/backrest/issues/138)) ([79715a9](https://github.com/garethgeorge/backrest/commit/79715a97b34af60ca90894065d89c9ae603f0a59))
- proper display of retention policy ([38ff5fe](https://github.com/garethgeorge/backrest/commit/38ff5fecee3ff3cdff5c7ccecb48e600eb714511))
- properly parse repo flags ([348ec46](https://github.com/garethgeorge/backrest/commit/348ec4690cab74c3089f2be33d889df3002a5a97))
- stat operation interval for long running repos ([f2477ab](https://github.com/garethgeorge/backrest/commit/f2477ab06cbe571723cd7290e06e8890747f81aa))
- stats chart titles invisible on light color theme ([746fd9c](https://github.com/garethgeorge/backrest/commit/746fd9cf768f0c87a25f0015bd20289716b08604))

### Miscellaneous Chores

- bump version to 0.15.0 ([db4b76d](https://github.com/garethgeorge/backrest/commit/db4b76de8ed09c9eda6216e8dfe041518f5bbfc5))

## [0.13.0](https://github.com/garethgeorge/backrest/compare/v0.12.2...v0.13.0) (2024-02-21)

### Features

- add case insensitive excludes (iexcludes) ([#108](https://github.com/garethgeorge/backrest/issues/108)) ([bf6fb7e](https://github.com/garethgeorge/backrest/commit/bf6fb7e71402590961271e91ad6da63db27ff5ad))
- add flags to configure backrest options e.g. --config-file, --data-dir, --restic-cmd, --bind-address ([41ddc8e](https://github.com/garethgeorge/backrest/commit/41ddc8e1a9d5501a92498c8cf3c72625bd181f8a))
- add opt-in auto-unlock feature to remove locks on forget and prune ([#107](https://github.com/garethgeorge/backrest/issues/107)) ([c1ee33f](https://github.com/garethgeorge/backrest/commit/c1ee33f0cd65a23ec0090852ee0fc5fa50e72b64))
- add rclone binary to docker image and arm64 support ([#105](https://github.com/garethgeorge/backrest/issues/105)) ([5a49f2f](https://github.com/garethgeorge/backrest/commit/5a49f2f063e887cba85bba0347ebce3efe15753e))
- bundle rclone, busybox commands, and bash in default backrest docker image ([cec04f8](https://github.com/garethgeorge/backrest/commit/cec04f8f745d4bcfd49829c43367c61cb9778174))
- display non-fatal errors in backup operations (e.g. unreadable files) in UI ([#100](https://github.com/garethgeorge/backrest/issues/100)) ([caac35a](https://github.com/garethgeorge/backrest/commit/caac35a5402d056b626b59d19084d6a699d4346d))

### Bug Fixes

- improve error message when rclone config is missing ([663b430](https://github.com/garethgeorge/backrest/commit/663b430598e0890df74989af12ae81fae7922251))
- improved sidebar status refresh interval during live operations ([3d192fd](https://github.com/garethgeorge/backrest/commit/3d192fd59d98c242ed583d00eeec37e68a4a2ff5))
- live backup progress updates with partial-backup errors ([97a4948](https://github.com/garethgeorge/backrest/commit/97a494847ac5031866c31db0bb32219e6b2a0038))
- migrate prune policy options to oneof ([ef41d34](https://github.com/garethgeorge/backrest/commit/ef41d34d5312b6a3bcc4af536f64275cd20da657))
- restore operations should succeed for unassociated snapshots ([448107d](https://github.com/garethgeorge/backrest/commit/448107d22612f040fd45493246088277a4a72f63))
- separate docker images for scratch and alpine linux base ([#106](https://github.com/garethgeorge/backrest/issues/106)) ([40e3e04](https://github.com/garethgeorge/backrest/commit/40e3e04a686f0a1749fa39e15821e6310e0ccf52))

## [0.12.2](https://github.com/garethgeorge/backrest/compare/v0.12.1...v0.12.2) (2024-02-16)

### Bug Fixes

- release-please automation ([63ddf15](https://github.com/garethgeorge/backrest/commit/63ddf15bf9799de30bda8548421e11e1bcd9ed05))

## [0.12.1](https://github.com/garethgeorge/backrest/compare/v0.12.0...v0.12.1) (2024-02-16)

### Bug Fixes

- delete event button in UI is hard to see on light theme ([8a05df8](https://github.com/garethgeorge/backrest/commit/8a05df87fcc44699c890f0cbe1065d79f49e1cc2))
- use 'embed' to package WebUI sources instead of go.rice ([e3ba5cf](https://github.com/garethgeorge/backrest/commit/e3ba5cf12ebfedafaa2125687bd7522f29ccab51))

## [0.12.0](https://github.com/garethgeorge/backrest/compare/v0.11.1...v0.12.0) (2024-02-15)

### Features

- add button to forget individual snapshots ([276b1d2](https://github.com/garethgeorge/backrest/commit/276b1d2c602ad0f787958452070771af3e69f073))
- add slack webhook ([8fa90ab](https://github.com/garethgeorge/backrest/commit/8fa90ab9ca48f0888ed0a5d263cb697758063188))
- Add support for multiple sets of expected env vars per repo scheme ([#90](https://github.com/garethgeorge/backrest/issues/90)) ([da0551c](https://github.com/garethgeorge/backrest/commit/da0551c19a98fe675d278e34f8e3cc58ac9edaf5))
- clear operations from history ([dc7a3a5](https://github.com/garethgeorge/backrest/commit/dc7a3a59a2400f97dd6b8140c6e70a34105496f9))
- Windows WebUI uses correct path separator ([f5521e7](https://github.com/garethgeorge/backrest/commit/f5521e7b56e446fa2062a95560f315621b77d3e6))

### Bug Fixes

- cleanup old versions of restic when upgrading ([79f529f](https://github.com/garethgeorge/backrest/commit/79f529f8edfb9bf893e74f7b1355bd7f2d7bdc3f))
- hide delete operation button if operation is in progress or pending ([08c8762](https://github.com/garethgeorge/backrest/commit/08c876243febb99a68740c449055e850f37d740e))
- retention policy configuration in add plan view ([dd24d90](https://github.com/garethgeorge/backrest/commit/dd24d9024f5ade62535956b1449dae75627ce493))
- stats operations running at wrong interval ([05e5ae0](https://github.com/garethgeorge/backrest/commit/05e5ae0c455680bf9fbc9b4b2a9fbf96bcfdfc3b))

## [0.11.1](https://github.com/garethgeorge/backrest/compare/v0.11.0...v0.11.1) (2024-02-08)

### Bug Fixes

- backrest fails to create directory for jwt secrets ([0067edf](https://github.com/garethgeorge/backrest/commit/0067edf378b01147f0041c225994098cb9c452ab))
- form bugs in UI e.g. awkward behavior when modifying hooks ([4fcf526](https://github.com/garethgeorge/backrest/commit/4fcf52602c114e2c639fc4302a9b8e8d51180a4d))
- update restic version to 1.16.4 ([668a7cb](https://github.com/garethgeorge/backrest/commit/668a7cb5bb5c0955a0e3186b2dd9329cedddd96f))
- wrong field names in hooks form ([3540904](https://github.com/garethgeorge/backrest/commit/354090497b73d40d8a9e705d1aa0c4662ffc4b0e))
- wrong value passed to --max-unused when providing a custom prune policy ([34175f2](https://github.com/garethgeorge/backrest/commit/34175f273630f7d2324a4d6b5f9f2f7576dd6608))

## [0.11.0](https://github.com/garethgeorge/backrest/compare/v0.10.1...v0.11.0) (2024-02-04)

### Features

- add user configurable command hooks for backup lifecycle events ([#60](https://github.com/garethgeorge/backrest/issues/60)) ([9be413b](https://github.com/garethgeorge/backrest/commit/9be413bbcca796862f161a769991ab695a50b464))
- authentication for WebUI ([#62](https://github.com/garethgeorge/backrest/issues/62)) ([4a1f326](https://github.com/garethgeorge/backrest/commit/4a1f3268a7de0533e0a979b9e97a7117b028358e))
- implement discord hook type ([25924b6](https://github.com/garethgeorge/backrest/commit/25924b6197c870f9dfc1e04f5be39377251e7f2d))
- implement gotify hook type ([e0ce655](https://github.com/garethgeorge/backrest/commit/e0ce6558c047f3aff068ee5d475fa1bdba380c4d))
- support keep-all retention policy for append-only backups ([f163c02](https://github.com/garethgeorge/backrest/commit/f163c02d7d2c798b4057037a996de44e34de9f2b))

### Bug Fixes

- add API test coverage and fix minor bugs ([f5bb74b](https://github.com/garethgeorge/backrest/commit/f5bb74bf246fcd5712dbbc85f4233169f7db4aa7))
- add first time setup hint for user authentication ([4a565f2](https://github.com/garethgeorge/backrest/commit/4a565f2cdcd091e0eabc302ab91e53012f53eb26))
- add test coverage for log rotation ([f1084ca](https://github.com/garethgeorge/backrest/commit/f1084cab4894751ba4a92f9be6b6b70d9084d0e6))
- bugfixes for auth flow ([427792c](https://github.com/garethgeorge/backrest/commit/427792c7244fb712bbea0557d4a6c7ee07052534))
- stats not displaying on long running repos ([f1ba1d9](https://github.com/garethgeorge/backrest/commit/f1ba1d91f37234f24ae5202d27114a33432366da))
- store large log outputs in tar bundles of logs ([0cf01e0](https://github.com/garethgeorge/backrest/commit/0cf01e020640b0145bcd0d25a38cde1fce940aff))
- windows install errors on decompressing zip archive ([5323b9f](https://github.com/garethgeorge/backrest/commit/5323b9ffc065bc3b28171575cdccc4358b69750b))

## [0.10.1](https://github.com/garethgeorge/backrest/compare/v0.10.0...v0.10.1) (2024-01-25)

### Bug Fixes

- chmod config 0600 such that only the creating user can read ([ecff0e5](https://github.com/garethgeorge/backrest/commit/ecff0e57c1fa4d65f35774d227a27222af8e7921))
- install scripts handle working dir correctly ([dcff2ad](https://github.com/garethgeorge/backrest/commit/dcff2adf60222030043d7a227d27e74f555ab376))
- relax name regex for plans and repos ([ee6134a](https://github.com/garethgeorge/backrest/commit/ee6134af76c3e90f542f67b89b2571f060db5590))
- sftp support using public key authentication ([bedb302](https://github.com/garethgeorge/backrest/commit/bedb302a025438a58309f26b046c9b6d49316414))
- typos in validation error messages in addrepomodel ([3b79afb](https://github.com/garethgeorge/backrest/commit/3b79afb2b18530deaa10cca08a60941a64c6fd9b))

## [0.10.0](https://github.com/garethgeorge/backrest/compare/v0.9.3...v0.10.0) (2024-01-15)

### Features

- make prune policy configurable in the addrepoview in the UI ([3fd08eb](https://github.com/garethgeorge/backrest/commit/3fd08eb8e4b455db656a0680318851824fdad2db))
- update restic dependency to v0.16.3 ([ac8db31](https://github.com/garethgeorge/backrest/commit/ac8db31713d4db3c2240b7f7c006e518e9e0726c))
- verify gpg signature when downloading and installing restic binary ([04106d1](https://github.com/garethgeorge/backrest/commit/04106d15d5ad73db6e670e84340ac1f9be200a23))

## [0.9.3](https://github.com/garethgeorge/backrest/compare/v0.9.2...v0.9.3) (2024-01-05)

### Bug Fixes

- correctly mark tasks as inprogress before execution ([b19438a](https://github.com/garethgeorge/backrest/commit/b19438afbd7b83dc964774347e64491143a3a5d2))
- correctly select light/dark mode based on system colortheme ([b64199c](https://github.com/garethgeorge/backrest/commit/b64199c140db7d2a77b58219cee088d22ec81954))

## [0.9.2](https://github.com/garethgeorge/backrest/compare/v0.9.1...v0.9.2) (2024-01-01)

### Bug Fixes

- possible race condition in scheduled task heap ([30874c9](https://github.com/garethgeorge/backrest/commit/30874c9150f32a0fba5f1ea99bc77bcc978d8b03))

## [0.9.1](https://github.com/garethgeorge/backrest/compare/v0.9.0...v0.9.1) (2024-01-01)

### Bug Fixes

- failed forget operations are hidden in the UI ([9896446](https://github.com/garethgeorge/backrest/commit/9896446ccfbcb8475a21b5fb565ebb73cb6bac2c))
- UI buttons spin while waiting for tasks to complete ([c767fa7](https://github.com/garethgeorge/backrest/commit/c767fa7476d76f1b4eb49443a19ee1cedb4eb70a))

## [0.9.0](https://github.com/garethgeorge/backrest/compare/v0.8.1...v0.9.0) (2023-12-31)

### Features

- add backrest logo ([5add0d8](https://github.com/garethgeorge/backrest/commit/5add0d8ffa829a71103520c94eacae17966f2a9f))
- add mobile layout ([9c7f227](https://github.com/garethgeorge/backrest/commit/9c7f227ad0f5df34d66390c94b64e9f5181d24f0))
- index snapshots created outside of backrest ([7711297](https://github.com/garethgeorge/backrest/commit/7711297a84170a733c5ccdb3e89617efc878cf69))
- schedule index operations and stats refresh from repo view ([851bd12](https://github.com/garethgeorge/backrest/commit/851bd125b640e65a5b98b67d28d2f29e94411646))

### Bug Fixes

- operations associated with incorrect ID when tasks are rescheduled ([25871c9](https://github.com/garethgeorge/backrest/commit/25871c99920d8717e91bf1a921109b9df82a59a1))
- reduce stats refresh frequency ([adbe005](https://github.com/garethgeorge/backrest/commit/adbe0056d82a5d9f890ce79b1120f5084bdc7124))
- stat never runs ([3f3252d](https://github.com/garethgeorge/backrest/commit/3f3252d47951270fbf5f21b0831effb121d3ba3f))
- stats task priority ([6bfe769](https://github.com/garethgeorge/backrest/commit/6bfe769fe037a5f2d35947574a5ed7e26ba981a8))
- tasks run late when laptops resume from sleep ([cb78298](https://github.com/garethgeorge/backrest/commit/cb78298cffb492560717d5f8bdcd5941f7976f2e))
- UI and code quality improvements ([c5e435d](https://github.com/garethgeorge/backrest/commit/c5e435d640bc8e79ceacf7f64d4cf75644859204))

## [0.8.0](https://github.com/garethgeorge/backrest/compare/v0.7.0...v0.8.0) (2023-12-25)

### Features

- add repo stats to restic package ([26d4724](https://github.com/garethgeorge/backrest/commit/26d47243c1e31f17c4d8adc6227325551854ce1f))
- add stats to repo view e.g. total size in storage ([adb0e3f](https://github.com/garethgeorge/backrest/commit/adb0e3f23050a86cd1c507d374e9d45f5eb5ee27))
- display last operation status for each plan and repo in UI ([cc11197](https://github.com/garethgeorge/backrest/commit/cc111970ca2e61cf39804378808aa5b5f77f9581))

### Bug Fixes

- crashing bug on partial backup ([#39](https://github.com/garethgeorge/backrest/issues/39)) ([fba6c8d](https://github.com/garethgeorge/backrest/commit/fba6c8da869d66b7b44f87a0dc1e3779924c31b7))
- install scripts and improved asset compression ([b8c2e81](https://github.com/garethgeorge/backrest/commit/b8c2e813586f2b48c78d70e09a29c5052621caf1))

## [0.7.0](https://github.com/garethgeorge/backrest/compare/v0.6.0...v0.7.0) (2023-12-22)

### Features

- add activity bar to UI heading ([f5c3e76](https://github.com/garethgeorge/backrest/commit/f5c3e762ed4ed3c908e843d74985fb6c7b253db7))
- add clear error history button ([48d80b9](https://github.com/garethgeorge/backrest/commit/48d80b9473db6619518924d0849b0eda78e62afa))
- add repo view ([9522ac1](https://github.com/garethgeorge/backrest/commit/9522ac18deedc15311d3d464ee36c20e7f72e39f))
- autoinstall required restic version ([b385c01](https://github.com/garethgeorge/backrest/commit/b385c011210087e6d6992a4e4b279fec4b22ab89))
- basic forget support in backend and UI ([d22d9d1](https://github.com/garethgeorge/backrest/commit/d22d9d1a05831fae94ce397c0c73c6292d378cf5))
- begin UI integration with backend ([cccdd29](https://github.com/garethgeorge/backrest/commit/cccdd297c15cd47268b2a1903e9624bdbca3dc68))
- display queued operations ([0c818bb](https://github.com/garethgeorge/backrest/commit/0c818bb9452a944d8b1127e553142e1e60ed90af))
- forget soft-deletes operations associated with removed snapshots ([f3dc7ff](https://github.com/garethgeorge/backrest/commit/f3dc7ffd077fef67870852f8f4e8b9aa6c94806e))
- forget soft-deletes operations associated with removed snapshots ([38bc107](https://github.com/garethgeorge/backrest/commit/38bc107db394716e34245f1edefc5e4cf4a15333))
- implement add plan UI ([9288589](https://github.com/garethgeorge/backrest/commit/92885898cf551a2dcb4bb315f130138cd7a8cc67))
- implement backup scheduling in orchestrator ([eadb1a8](https://github.com/garethgeorge/backrest/commit/eadb1a82019f0cfc82edf8559adbad7730a4e86a))
- implement basic plan view ([4c6f042](https://github.com/garethgeorge/backrest/commit/4c6f042250946a036e46225e669ee39e2433b198))
- implement delete button for plan and repo UIs ([ffb0d85](https://github.com/garethgeorge/backrest/commit/ffb0d859f19f4af66a7521768dab083995f9672a))
- implement forget and prune support in restic pkg ([ffb4573](https://github.com/garethgeorge/backrest/commit/ffb4573737a73cc32f325bc0b9c3feed764b7879))
- implement forget operation ([ebccf3b](https://github.com/garethgeorge/backrest/commit/ebccf3bc3b78083aee635de7c6ae23b52ee88284))
- implement garbage collection of old operations ([46456a8](https://github.com/garethgeorge/backrest/commit/46456a88870934506ede4b67c3dfaa2f2afcee14))
- implement prune support ([#25](https://github.com/garethgeorge/backrest/issues/25)) ([a311b0a](https://github.com/garethgeorge/backrest/commit/a311b0a3fb5315f17d66361a3e72fa10b8a744a1))
- implement repo unlocking and operation list implementation ([6665ad9](https://github.com/garethgeorge/backrest/commit/6665ad98d7f54bea30ea532932a8a3409717c913))
- implement repo, edit, and supporting RPCs ([d282c32](https://github.com/garethgeorge/backrest/commit/d282c32c8bd3d8f5747e934d4af6a84faca1ec86))
- implement restore operation through snapshot browser UI ([#27](https://github.com/garethgeorge/backrest/issues/27)) ([d758509](https://github.com/garethgeorge/backrest/commit/d758509797e21e3ec4bc67eff4d974604e4a5476))
- implement snapshot browsing ([8ffffa0](https://github.com/garethgeorge/backrest/commit/8ffffa05e41ca31e2d38fde5427dae34ac4a1abb))
- implement snapshot indexing ([a90b30e](https://github.com/garethgeorge/backrest/commit/a90b30e19f7107874bbfe244451b07f72c437213))
- improve oplist performance and display forget operations in oplist ([#22](https://github.com/garethgeorge/backrest/issues/22)) ([51b4921](https://github.com/garethgeorge/backrest/commit/51b49214e3d32cc4b28e13085bd196ba164a8c19))
- initial oplog implementation ([dd9142c](https://github.com/garethgeorge/backrest/commit/dd9142c0e97e1175ff12f2861220af0e0d68b7d9))
- initial optree implementation ([ba390a2](https://github.com/garethgeorge/backrest/commit/ba390a2ca1b5e9adaab36a7db0d988f54f5a6cdd))
- initial Windows OS support ([f048cbf](https://github.com/garethgeorge/backrest/commit/f048cbf10dc60da51cd7f5aee4614a8750fd85b2))
- match system color theme (darkmode support) ([a8762dc](https://github.com/garethgeorge/backrest/commit/a8762dca329927b93db40b01cc011c00e12891f0))
- operations IDs are ordered by operation timestamp ([a1ed6f9](https://github.com/garethgeorge/backrest/commit/a1ed6f90ba1d608e00c53221db45b67251085aa7))
- present list of operations on plan view ([6491dbe](https://github.com/garethgeorge/backrest/commit/6491dbed146967c0e12eee4392d1d12843dc7c5e))
- repo can be created through UI ([9ccade5](https://github.com/garethgeorge/backrest/commit/9ccade5ccd97f4e485d52ad5c675be6b0a4a1049))
- scaffolding basic UI structure ([1273f81](https://github.com/garethgeorge/backrest/commit/1273f8105a2549b0ccd0c7a588eb60646b66366e))
- show snapshots in sidenav ([1a9a5b6](https://github.com/garethgeorge/backrest/commit/1a9a5b60d24dd75752e5a3f84dd87af3e38422bb))
- snapshot items are viewable in the UI and minor element ordering fixes ([a333001](https://github.com/garethgeorge/backrest/commit/a33300175c645f31b95b3038de02821a1f3d5559))
- support ImportSnapshotOperation in oplog ([89f95b3](https://github.com/garethgeorge/backrest/commit/89f95b351fe250534cd39ac27ff34b2b148256e1))
- support task cancellation ([fc9c06d](https://github.com/garethgeorge/backrest/commit/fc9c06df00409b73dda23f4be031746f492b1a24))
- update getting started guide ([2c421d6](https://github.com/garethgeorge/backrest/commit/2c421d661501fa4a3120aa3f39937cd58b29c2dc))

### Bug Fixes

- backup ordering in tree view ([b9c8b3e](https://github.com/garethgeorge/backrest/commit/b9c8b3e378e88a0feff4d477d9d97eb5db075382))
- build and test fixes ([4957496](https://github.com/garethgeorge/backrest/commit/49574967871494dcb5095e5699610097466f57f9))
- connectivity issues with embedded server ([482cc8e](https://github.com/garethgeorge/backrest/commit/482cc8ebbc93b919991f6566b212247c5874f70f))
- deadlock in snapshots ([93b2120](https://github.com/garethgeorge/backrest/commit/93b2120f74ea348e5084ab430573368bf4066eec))
- forget deadlocking and misc smaller bugs ([b7c633d](https://github.com/garethgeorge/backrest/commit/b7c633d021d68d4880a5f442ce70a858002b4af2))
- hide no-op prune operations ([20dd78c](https://github.com/garethgeorge/backrest/commit/20dd78cac4bdd6385cb7a0ea9ff0be75fde9135b))
- improve error message formatting ([ae33b01](https://github.com/garethgeorge/backrest/commit/ae33b01de408af3b1d711a369298a2782a24ad1e))
- improve operation ordering to fix snapshots indexed before forget operation ([#21](https://github.com/garethgeorge/backrest/issues/21)) ([b513b08](https://github.com/garethgeorge/backrest/commit/b513b08e51434c28c90f5f062b4ae292f6854f4e))
- improve output detail collection for command failures ([c492f9b](https://github.com/garethgeorge/backrest/commit/c492f9ba63169942509349797ebe951879b53635))
- improve UI performance ([8488d46](https://github.com/garethgeorge/backrest/commit/8488d461bd7ffec2e8171d67f83093c32c79073f))
- improve Windows path handling ([426aad4](https://github.com/garethgeorge/backrest/commit/426aad4890d2de5d70cd2e0232c0d11c42606c92))
- incorrrect handling of oplog events in UI ([95ca96a](https://github.com/garethgeorge/backrest/commit/95ca96a31f2e1ead2702164ec8675e4b4f54cf1d))
- operation ordering in tree view ([2b4e1a2](https://github.com/garethgeorge/backrest/commit/2b4e1a2fdbf11b010ddbcd0b6fd2640d01e4dbc8))
- operations marked as 'warn' rather than 'error' for partial backups ([fe92b62](https://github.com/garethgeorge/backrest/commit/fe92b625780481193e0ab63fbbdddb889bbda2a8))
- ordering of operations when viewed from backup tree ([063f086](https://github.com/garethgeorge/backrest/commit/063f086a6e31df250dd9be42cdb5fa549307106f))
- race condition in snapshot browser ([f239b91](https://github.com/garethgeorge/backrest/commit/f239b9170415e063ec8d60a5b5e14ae3610b9bad))
- relax output parsing to skip over warnings ([8f85b74](https://github.com/garethgeorge/backrest/commit/8f85b747f57844bbc898668723eec50a1666aa39))
- repo orchestrator tests ([d077fc8](https://github.com/garethgeorge/backrest/commit/d077fc83c97b7fbdbeda9702828c8780182b2616))
- restic fails to detect summary event for very short backups ([46b2a85](https://github.com/garethgeorge/backrest/commit/46b2a8567706ddb21cfcf3e18b57e16d50809b56))
- restic should initialize repo on backup operation ([e57abbd](https://github.com/garethgeorge/backrest/commit/e57abbdcb1864c362e6ae3c22850c0380671cb34))
- restora should not init repos added manually e.g. without the UI ([68b50e1](https://github.com/garethgeorge/backrest/commit/68b50e1eb5a2ebd861c869f71f49d196cb5214f8))
- snapshots out of order in UI ([b9bcc7e](https://github.com/garethgeorge/backrest/commit/b9bcc7e7c758abafa4878b6ef895adf2d2d0bc42))
- standardize on fully qualified snapshot_id and decouple protobufs from restic package ([e6031bf](https://github.com/garethgeorge/backrest/commit/e6031bfa543a7300e622c1b0f56efc6320e7611e))
- support more versions of restic ([0cdfd11](https://github.com/garethgeorge/backrest/commit/0cdfd115e29a0b08d5814e71c0f4a8f2baf52e90))
- task cancellation ([d49b729](https://github.com/garethgeorge/backrest/commit/d49b72996ea7fd0543d55db3fc8e1127fe5a2476))
- task priority not taking effect ([af7462c](https://github.com/garethgeorge/backrest/commit/af7462cefb130153cdaaa08e8ebefefa40e80e49))
- time formatting in operation list ([53c7f12](https://github.com/garethgeorge/backrest/commit/53c7f1248f5284080fff872ac79b3996474412b3))
- UI layout adjustments ([7d1b95c](https://github.com/garethgeorge/backrest/commit/7d1b95c81f0f69840ce1d20cb0d4a4bb90011dc9))
- unexpected config location on MacOS ([8d40576](https://github.com/garethgeorge/backrest/commit/8d40576c6526d6f180c96fbeb81d7f59f56b51b8))
- use timezone offset when grouping operations in OperationTree ([06240bd](https://github.com/garethgeorge/backrest/commit/06240bd7adabd76424025030cfde2fb5e54c219f))

## [0.6.0](https://github.com/garethgeorge/backrest/compare/v0.5.0...v0.6.0) (2023-12-22)

### Features

- add activity bar to UI heading ([f5c3e76](https://github.com/garethgeorge/backrest/commit/f5c3e762ed4ed3c908e843d74985fb6c7b253db7))
- add clear error history button ([48d80b9](https://github.com/garethgeorge/backrest/commit/48d80b9473db6619518924d0849b0eda78e62afa))
- add repo view ([9522ac1](https://github.com/garethgeorge/backrest/commit/9522ac18deedc15311d3d464ee36c20e7f72e39f))
- implement garbage collection of old operations ([46456a8](https://github.com/garethgeorge/backrest/commit/46456a88870934506ede4b67c3dfaa2f2afcee14))
- support task cancellation ([fc9c06d](https://github.com/garethgeorge/backrest/commit/fc9c06df00409b73dda23f4be031746f492b1a24))

### Bug Fixes

- backup ordering in tree view ([b9c8b3e](https://github.com/garethgeorge/backrest/commit/b9c8b3e378e88a0feff4d477d9d97eb5db075382))
- hide no-op prune operations ([20dd78c](https://github.com/garethgeorge/backrest/commit/20dd78cac4bdd6385cb7a0ea9ff0be75fde9135b))
- incorrrect handling of oplog events in UI ([95ca96a](https://github.com/garethgeorge/backrest/commit/95ca96a31f2e1ead2702164ec8675e4b4f54cf1d))
- operation ordering in tree view ([2b4e1a2](https://github.com/garethgeorge/backrest/commit/2b4e1a2fdbf11b010ddbcd0b6fd2640d01e4dbc8))
- operations marked as 'warn' rather than 'error' for partial backups ([fe92b62](https://github.com/garethgeorge/backrest/commit/fe92b625780481193e0ab63fbbdddb889bbda2a8))
- race condition in snapshot browser ([f239b91](https://github.com/garethgeorge/backrest/commit/f239b9170415e063ec8d60a5b5e14ae3610b9bad))
- restic should initialize repo on backup operation ([e57abbd](https://github.com/garethgeorge/backrest/commit/e57abbdcb1864c362e6ae3c22850c0380671cb34))
- backrest should not init repos added manually e.g. without the UI ([68b50e1](https://github.com/garethgeorge/backrest/commit/68b50e1eb5a2ebd861c869f71f49d196cb5214f8))
- task cancellation ([d49b729](https://github.com/garethgeorge/backrest/commit/d49b72996ea7fd0543d55db3fc8e1127fe5a2476))
- use timezone offset when grouping operations in OperationTree ([06240bd](https://github.com/garethgeorge/backrest/commit/06240bd7adabd76424025030cfde2fb5e54c219f))

## [0.5.0](https://github.com/garethgeorge/backrest/compare/v0.4.0...v0.5.0) (2023-12-10)

### Features

- implement repo unlocking and operation list implementation ([6665ad9](https://github.com/garethgeorge/backrest/commit/6665ad98d7f54bea30ea532932a8a3409717c913))
- initial Windows OS support ([f048cbf](https://github.com/garethgeorge/backrest/commit/f048cbf10dc60da51cd7f5aee4614a8750fd85b2))
- match system color theme (darkmode support) ([a8762dc](https://github.com/garethgeorge/backrest/commit/a8762dca329927b93db40b01cc011c00e12891f0))

### Bug Fixes

- improve output detail collection for command failures ([c492f9b](https://github.com/garethgeorge/backrest/commit/c492f9ba63169942509349797ebe951879b53635))
- improve Windows path handling ([426aad4](https://github.com/garethgeorge/backrest/commit/426aad4890d2de5d70cd2e0232c0d11c42606c92))
- ordering of operations when viewed from backup tree ([063f086](https://github.com/garethgeorge/backrest/commit/063f086a6e31df250dd9be42cdb5fa549307106f))
- relax output parsing to skip over warnings ([8f85b74](https://github.com/garethgeorge/backrest/commit/8f85b747f57844bbc898668723eec50a1666aa39))
- snapshots out of order in UI ([b9bcc7e](https://github.com/garethgeorge/backrest/commit/b9bcc7e7c758abafa4878b6ef895adf2d2d0bc42))
- unexpected config location on MacOS ([8d40576](https://github.com/garethgeorge/backrest/commit/8d40576c6526d6f180c96fbeb81d7f59f56b51b8))

## [0.4.0](https://github.com/garethgeorge/backrest/compare/v0.3.0...v0.4.0) (2023-12-04)

### Features

- implement prune support ([#25](https://github.com/garethgeorge/backrest/issues/25)) ([a311b0a](https://github.com/garethgeorge/backrest/commit/a311b0a3fb5315f17d66361a3e72fa10b8a744a1))
- implement restore operation through snapshot browser UI ([#27](https://github.com/garethgeorge/backrest/issues/27)) ([d758509](https://github.com/garethgeorge/backrest/commit/d758509797e21e3ec4bc67eff4d974604e4a5476))

## [0.3.0](https://github.com/garethgeorge/backrest/compare/v0.2.0...v0.3.0) (2023-12-03)

### Features

- autoinstall required restic version ([b385c01](https://github.com/garethgeorge/backrest/commit/b385c011210087e6d6992a4e4b279fec4b22ab89))
- basic forget support in backend and UI ([d22d9d1](https://github.com/garethgeorge/backrest/commit/d22d9d1a05831fae94ce397c0c73c6292d378cf5))
- begin UI integration with backend ([cccdd29](https://github.com/garethgeorge/backrest/commit/cccdd297c15cd47268b2a1903e9624bdbca3dc68))
- display queued operations ([0c818bb](https://github.com/garethgeorge/backrest/commit/0c818bb9452a944d8b1127e553142e1e60ed90af))
- forget soft-deletes operations associated with removed snapshots ([f3dc7ff](https://github.com/garethgeorge/backrest/commit/f3dc7ffd077fef67870852f8f4e8b9aa6c94806e))
- forget soft-deletes operations associated with removed snapshots ([38bc107](https://github.com/garethgeorge/backrest/commit/38bc107db394716e34245f1edefc5e4cf4a15333))
- implement add plan UI ([9288589](https://github.com/garethgeorge/backrest/commit/92885898cf551a2dcb4bb315f130138cd7a8cc67))
- implement backup scheduling in orchestrator ([eadb1a8](https://github.com/garethgeorge/backrest/commit/eadb1a82019f0cfc82edf8559adbad7730a4e86a))
- implement basic plan view ([4c6f042](https://github.com/garethgeorge/backrest/commit/4c6f042250946a036e46225e669ee39e2433b198))
- implement delete button for plan and repo UIs ([ffb0d85](https://github.com/garethgeorge/backrest/commit/ffb0d859f19f4af66a7521768dab083995f9672a))
- implement forget and prune support in restic pkg ([ffb4573](https://github.com/garethgeorge/backrest/commit/ffb4573737a73cc32f325bc0b9c3feed764b7879))
- implement forget operation ([ebccf3b](https://github.com/garethgeorge/backrest/commit/ebccf3bc3b78083aee635de7c6ae23b52ee88284))
- implement repo, edit, and supporting RPCs ([d282c32](https://github.com/garethgeorge/backrest/commit/d282c32c8bd3d8f5747e934d4af6a84faca1ec86))
- implement snapshot browsing ([8ffffa0](https://github.com/garethgeorge/backrest/commit/8ffffa05e41ca31e2d38fde5427dae34ac4a1abb))
- implement snapshot indexing ([a90b30e](https://github.com/garethgeorge/backrest/commit/a90b30e19f7107874bbfe244451b07f72c437213))
- improve oplist performance and display forget operations in oplist ([#22](https://github.com/garethgeorge/backrest/issues/22)) ([51b4921](https://github.com/garethgeorge/backrest/commit/51b49214e3d32cc4b28e13085bd196ba164a8c19))
- initial oplog implementation ([dd9142c](https://github.com/garethgeorge/backrest/commit/dd9142c0e97e1175ff12f2861220af0e0d68b7d9))
- initial optree implementation ([ba390a2](https://github.com/garethgeorge/backrest/commit/ba390a2ca1b5e9adaab36a7db0d988f54f5a6cdd))
- operations IDs are ordered by operation timestamp ([a1ed6f9](https://github.com/garethgeorge/backrest/commit/a1ed6f90ba1d608e00c53221db45b67251085aa7))
- present list of operations on plan view ([6491dbe](https://github.com/garethgeorge/backrest/commit/6491dbed146967c0e12eee4392d1d12843dc7c5e))
- repo can be created through UI ([9ccade5](https://github.com/garethgeorge/backrest/commit/9ccade5ccd97f4e485d52ad5c675be6b0a4a1049))
- scaffolding basic UI structure ([1273f81](https://github.com/garethgeorge/backrest/commit/1273f8105a2549b0ccd0c7a588eb60646b66366e))
- show snapshots in sidenav ([1a9a5b6](https://github.com/garethgeorge/backrest/commit/1a9a5b60d24dd75752e5a3f84dd87af3e38422bb))
- snapshot items are viewable in the UI and minor element ordering fixes ([a333001](https://github.com/garethgeorge/backrest/commit/a33300175c645f31b95b3038de02821a1f3d5559))
- support ImportSnapshotOperation in oplog ([89f95b3](https://github.com/garethgeorge/backrest/commit/89f95b351fe250534cd39ac27ff34b2b148256e1))
- update getting started guide ([2c421d6](https://github.com/garethgeorge/backrest/commit/2c421d661501fa4a3120aa3f39937cd58b29c2dc))

### Bug Fixes

- build and test fixes ([4957496](https://github.com/garethgeorge/backrest/commit/49574967871494dcb5095e5699610097466f57f9))
- connectivity issues with embedded server ([482cc8e](https://github.com/garethgeorge/backrest/commit/482cc8ebbc93b919991f6566b212247c5874f70f))
- deadlock in snapshots ([93b2120](https://github.com/garethgeorge/backrest/commit/93b2120f74ea348e5084ab430573368bf4066eec))
- forget deadlocking and misc smaller bugs ([b7c633d](https://github.com/garethgeorge/backrest/commit/b7c633d021d68d4880a5f442ce70a858002b4af2))
- improve error message formatting ([ae33b01](https://github.com/garethgeorge/backrest/commit/ae33b01de408af3b1d711a369298a2782a24ad1e))
- improve operation ordering to fix snapshots indexed before forget operation ([#21](https://github.com/garethgeorge/backrest/issues/21)) ([b513b08](https://github.com/garethgeorge/backrest/commit/b513b08e51434c28c90f5f062b4ae292f6854f4e))
- improve UI performance ([8488d46](https://github.com/garethgeorge/backrest/commit/8488d461bd7ffec2e8171d67f83093c32c79073f))
- repo orchestrator tests ([d077fc8](https://github.com/garethgeorge/backrest/commit/d077fc83c97b7fbdbeda9702828c8780182b2616))
- restic fails to detect summary event for very short backups ([46b2a85](https://github.com/garethgeorge/backrest/commit/46b2a8567706ddb21cfcf3e18b57e16d50809b56))
- standardize on fully qualified snapshot_id and decouple protobufs from restic package ([e6031bf](https://github.com/garethgeorge/backrest/commit/e6031bfa543a7300e622c1b0f56efc6320e7611e))
- support more versions of restic ([0cdfd11](https://github.com/garethgeorge/backrest/commit/0cdfd115e29a0b08d5814e71c0f4a8f2baf52e90))
- task priority not taking effect ([af7462c](https://github.com/garethgeorge/backrest/commit/af7462cefb130153cdaaa08e8ebefefa40e80e49))
- time formatting in operation list ([53c7f12](https://github.com/garethgeorge/backrest/commit/53c7f1248f5284080fff872ac79b3996474412b3))
- UI layout adjustments ([7d1b95c](https://github.com/garethgeorge/backrest/commit/7d1b95c81f0f69840ce1d20cb0d4a4bb90011dc9))

## [0.2.0](https://github.com/garethgeorge/backrest/compare/v0.1.3...v0.2.0) (2023-12-03)

### Features

- forget soft-deletes operations associated with removed snapshots ([f3dc7ff](https://github.com/garethgeorge/backrest/commit/f3dc7ffd077fef67870852f8f4e8b9aa6c94806e))
- forget soft-deletes operations associated with removed snapshots ([38bc107](https://github.com/garethgeorge/backrest/commit/38bc107db394716e34245f1edefc5e4cf4a15333))
- improve oplist performance and display forget operations in oplist ([#22](https://github.com/garethgeorge/backrest/issues/22)) ([51b4921](https://github.com/garethgeorge/backrest/commit/51b49214e3d32cc4b28e13085bd196ba164a8c19))

### Bug Fixes

- forget deadlocking and misc smaller bugs ([b7c633d](https://github.com/garethgeorge/backrest/commit/b7c633d021d68d4880a5f442ce70a858002b4af2))
- improve operation ordering to fix snapshots indexed before forget operation ([#21](https://github.com/garethgeorge/backrest/issues/21)) ([b513b08](https://github.com/garethgeorge/backrest/commit/b513b08e51434c28c90f5f062b4ae292f6854f4e))
- task priority not taking effect ([af7462c](https://github.com/garethgeorge/backrest/commit/af7462cefb130153cdaaa08e8ebefefa40e80e49))
- UI layout adjustments ([7d1b95c](https://github.com/garethgeorge/backrest/commit/7d1b95c81f0f69840ce1d20cb0d4a4bb90011dc9))

## [0.2.0](https://github.com/garethgeorge/backrest/compare/v0.1.3...v0.2.0) (2023-12-02)

### Features

- forget soft-deletes operations associated with removed snapshots ([f3dc7ff](https://github.com/garethgeorge/backrest/commit/f3dc7ffd077fef67870852f8f4e8b9aa6c94806e))
- forget soft-deletes operations associated with removed snapshots ([38bc107](https://github.com/garethgeorge/backrest/commit/38bc107db394716e34245f1edefc5e4cf4a15333))

### Bug Fixes

- forget deadlocking and misc smaller bugs ([b7c633d](https://github.com/garethgeorge/backrest/commit/b7c633d021d68d4880a5f442ce70a858002b4af2))
- improve operation ordering to fix snapshots indexed before forget operation ([#21](https://github.com/garethgeorge/backrest/issues/21)) ([b513b08](https://github.com/garethgeorge/backrest/commit/b513b08e51434c28c90f5f062b4ae292f6854f4e))
- task priority not taking effect ([af7462c](https://github.com/garethgeorge/backrest/commit/af7462cefb130153cdaaa08e8ebefefa40e80e49))
