use proto::{
    types::{self, Empty},
    v1rusticdaemon::{
        rustic_daemon_server::{RusticDaemon, RusticDaemonServer},
        RepoOpts, ValidateResponse, VersionResponse,
    },
};
use rustic_backend::BackendOptions;
use rustic_core::{
    BackupOptions, ConfigOptions, KeyOptions, PathList, Repository, RepositoryOptions,
    SnapshotOptions,
};
use simplelog::{Config, LevelFilter, SimpleLogger};
use std::{error::Error, pin::Pin, task::Context};
use tokio::{
    io::{AsyncRead, AsyncWrite},
    net::UnixListener,
};
use tonic::{
    transport::{server::Connected, Server},
    Request, Response, Status,
};

mod proto;

#[derive(Default)]
pub struct RusticDaemonImpl {}

impl RusticDaemonImpl {
    pub fn new() -> Self {
        Self::default()
    }

    pub fn parse_repo_opts(
        &self,
        repo_opts: RepoOpts,
    ) -> Result<RepositoryOptions, Box<dyn Error>> {
        let repo_opts = RepositoryOptions::default()
            .password(repo_opts.password)
            .to_owned();

        Ok(repo_opts)
    }
}

#[tonic::async_trait]
impl RusticDaemon for RusticDaemonImpl {
    async fn validate_repo_opts(
        &self,
        request: Request<RepoOpts>,
    ) -> Result<Response<ValidateResponse>, Status> {
        let repo_opts = request.into_inner();

        let mut errors: Vec<String> = vec![];
        let warnings: Vec<String> = vec![];

        match toml::from_str::<BackendOptions>(&repo_opts.backend_opts_toml) {
            Ok(_) => {}
            Err(e) => errors.push(format!("Invalid backend_opts_toml: {}", e)),
        }

        match toml::from_str::<RepositoryOptions>(&repo_opts.repo_opts_toml) {
            Ok(_) => {}
            Err(e) => errors.push(format!("Invalid repo_opts_toml: {}", e)),
        }

        return Ok(Response::new(ValidateResponse {
            errors: errors,
            warnings: warnings,
        }));
    }

    async fn exists(&self, request: Request<RepoOpts>) -> Result<(), Status> {
        let repo_opts = request.into_inner();

        return Ok(());
    }
}

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    let daemon_impl = RusticDaemonImpl::default();

    println!("Rustic Daemon server listening on 127.0.0.1:58834");
    Server::builder()
        .add_service(RusticDaemonServer::new(daemon_impl))
        .serve("127.0.0.1:50051".parse()?)
        .await?;

    // // Display info logs
    // let _ = SimpleLogger::init(LevelFilter::Info, Config::default());

    // // Display info logs
    // let _ = SimpleLogger::init(LevelFilter::Info, Config::default());

    // // Initialize Backends
    // let backends = BackendOptions::default()
    //     .repository("/tmp/repo")
    //     .to_backends()?;

    // // Init repository
    // let repo_opts = RepositoryOptions::default().password("test");
    // let key_opts = KeyOptions::default();
    // let config_opts: ConfigOptions = ConfigOptions::default();
    // let _repo = Repository::new(&repo_opts, &backends)?.init(&key_opts, &config_opts)?;

    // // Reopen
    // let repo = _repo.open()?.to_indexed_ids()?;

    // let backup_opts = BackupOptions::default();
    // let snap = SnapshotOptions::default().to_snapshot()?;
    // let path_list = PathList::from_string("/tmp/.ICE-unix")?.sanitize()?;

    // // Create snapshot
    // let snap = repo.backup(&backup_opts, &path_list, snap)?;

    // println!("Snapshot: {:?}", snap);

    Ok(())
}
