# Changelog

## [0.5.0](https://github.com/RHAM91/copy-tui/compare/v0.4.0...v0.5.0) (2026-11-27)


### Features

* update from the private catalog with signature verification ([751ed15](https://github.com/RHAM91/copy-tui/commit/751ed151ff94ba3638c3ff2ae7038b60f4479584))

## [0.4.0](https://github.com/RHAM91/copy-tui/compare/v0.3.0...v0.4.0) (2026-11-19)


### Features

* stack panels on narrow terminals and choose a theme with ctrl+t ([#14](https://github.com/RHAM91/copy-tui/issues/14)) ([7012ada](https://github.com/RHAM91/copy-tui/commit/7012ada871ec2ef98905ea191ede771477ab3953))

## [0.3.0](https://github.com/RHAM91/copy-tui/compare/v0.2.0...v0.3.0) (2026-11-19)


### Features

* open the same folder in both panels with = and ask before moving ([#12](https://github.com/RHAM91/copy-tui/issues/12)) ([5201386](https://github.com/RHAM91/copy-tui/commit/5201386ed5fd7e043c2db629ed54be6be236dc3b))

## [0.2.0](https://github.com/RHAM91/copy-tui/compare/v0.1.0...v0.2.0) (2026-11-18)


### Features

* save and open panel bookmarks with ctrl+s and ctrl+b ([#10](https://github.com/RHAM91/copy-tui/issues/10)) ([19ceb71](https://github.com/RHAM91/copy-tui/commit/19ceb716a5b1d562d8852f20b828e4b870908ce5))

## 0.1.0 (2026-11-17)


### Features

* add copiar update and a startup update notice ([584a115](https://github.com/RHAM91/copy-tui/commit/584a115bfc77c99449951cef86c4b6a508a54cdf))
* **fsops:** add cp -a style copy engine with progress and cancel ([41228d7](https://github.com/RHAM91/copy-tui/commit/41228d72a34bc159927c22e458bd82b90caab140))
* move selection to the other panel with x ([b8c8dd9](https://github.com/RHAM91/copy-tui/commit/b8c8dd99414f748368271b68cfa627b6430ee682))
* rename with r and create folders with a ([57f6ef7](https://github.com/RHAM91/copy-tui/commit/57f6ef7fc1c58ec9713ec5eb9faca3038a4d85b4))
* support macOS and Windows ([523147a](https://github.com/RHAM91/copy-tui/commit/523147a885e7b923b291154cf1e32185fdd6979a))
* trash with d and permanent delete with D ([06b6ebe](https://github.com/RHAM91/copy-tui/commit/06b6ebe0328a332db0566f3020f42da225276a64))
* **ui:** add help bar, full help overlay and README ([cf459a6](https://github.com/RHAM91/copy-tui/commit/cf459a6e737d7f6c0e68954aac0d8add7aadf1c8))
* **ui:** add incremental search and multi-selection ([433c743](https://github.com/RHAM91/copy-tui/commit/433c7438678a2b71b25ef3880d5af8f5bbfb7a10))
* **ui:** add Vim navigation and directory listing ([c003543](https://github.com/RHAM91/copy-tui/commit/c003543165ee210a75136e13497a2368feabfd13))
* **ui:** copy selection with space, conflict dialog and progress modal ([021ed31](https://github.com/RHAM91/copy-tui/commit/021ed31f122b26b59d292d868146ac6987076508))
* **ui:** scaffold dual-pane TUI with --no-icons flag ([71351cc](https://github.com/RHAM91/copy-tui/commit/71351cc935ad11030a0fb99e146eca483c36e746))


### Bug Fixes

* **fsops:** allow case-only renames and guard the Windows trash struct size ([ad14ba4](https://github.com/RHAM91/copy-tui/commit/ad14ba499c6b1cc4c6327e5f59709a6a4bd038cf))
* **fsops:** prevent data loss on overwrite, cancel and cross-device trash ([24926f5](https://github.com/RHAM91/copy-tui/commit/24926f568d8bb973e3d802ec4f13f4139c054319))
* **ui:** keep full status width when no update notice is shown ([302e57f](https://github.com/RHAM91/copy-tui/commit/302e57fe70fcb672cec773f80c9a5e54606932f8))
