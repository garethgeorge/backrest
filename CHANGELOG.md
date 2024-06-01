# Changelog

## [1.1.0](https://github.com/garethgeorge/backrest/compare/v1.0.0...v1.1.0) (2024-06-01)


### Features

* add windows installer and tray app ([#294](https://github.com/garethgeorge/backrest/issues/294)) ([8a7543c](https://github.com/garethgeorge/backrest/commit/8a7543c7bf7f245d87fa079c477c50b333dfba37))
* support nice/ionice as a repo setting ([#309](https://github.com/garethgeorge/backrest/issues/309)) ([0c9f366](https://github.com/garethgeorge/backrest/commit/0c9f366e439b57007259a2ca305ac00733638566))
* support restic check operation ([#303](https://github.com/garethgeorge/backrest/issues/303)) ([ce42f68](https://github.com/garethgeorge/backrest/commit/ce42f68d0d563defabbaafce63313fcf3835d2dd))


### Bug Fixes

* collection of ui refresh timing bugs ([b218bc9](https://github.com/garethgeorge/backrest/commit/b218bc9409bf4a6c70da06e1f98760ff520afc37))
* improve prune and check scheduling in new repos ([c58055e](https://github.com/garethgeorge/backrest/commit/c58055ec91ccc9a8afc5d3ff402f68da00a04e66))
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
