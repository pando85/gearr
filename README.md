<h1 align="center">
  <br>
  <img src="https://raw.githubusercontent.com/pando85/transcoder/master/server/web/ui/public/logo.svg" alt="logo" width="200">
  <br>
  Transcoder
  <br>
  <br>
</h1>

Transcoder is a program designed to operate on a server with two distinct types of agents for video
transcoding tasks, specifically converting a video library to the x265 format using ffmpeg. The
following information provides details on how to use and configure the Transcoder system.

## Container Images

- **Server:** `ghcr.io/pando85/transcoder:latest-server`
- **Worker:** `ghcr.io/pando85/transcoder:latest-worker`
- **PGS Worker:** `ghcr.io/pando85/transcoder:latest-worker-pgs`

## Configuration

### Environment Variables

The application supports configuration through environment variables. Below is a table of supported
environment variables and their default values:

#### Server

| Variable                 | Description                                           | Default Value         |
| ------------------------ | ----------------------------------------------------- | --------------------- |
| `BROKER_HOST`            | Broker host address                                   | localhost             |
| `BROKER_PORT`            | Broker port                                           | 5672                  |
| `BROKER_USER`            | Broker username                                       | broker                |
| `BROKER_PASSWORD`        | Broker password                                       | broker                |
| `BROKER_TASKENCODEQUEUE` | Broker tasks queue name for encoding                  | tasks                 |
| `BROKER_TASKPGSQUEUE`    | Broker tasks queue name for PGS to SRT conversion     | tasks_pgstosrt        |
| `BROKER_EVENTQUEUE`      | Broker tasks events queue name                        | task_events           |
| `DATABASE_DRIVER`        | Database driver                                       | postgres              |
| `DATABASE_HOST`          | Database host address                                 | localhost             |
| `DATABASE_PORT`          | Database port                                         | 5432                  |
| `DATABASE_USER`          | Database username                                     | postgres              |
| `DATABASE_PASSWORD`      | Database password                                     | postgres              |
| `DATABASE_DATABASE`      | Database name                                         | transcoder            |
| `DATABASE_SSLMODE`       | Database SSL mode                                     | disable               |
| `LOG_LEVEL`              | Log level (debug, info, warning, error, fatal)        | info                  |
| `SCHEDULER_DOMAIN`       | Base domain for worker downloads and uploads          | http://localhost:8080 |
| `SCHEDULER_SCHEDULETIME` | Scheduling loop execution interval                    | 5m                    |
| `SCHEDULER_JOBTIMEOUT`   | Requeue jobs running for more than specified duration | 24h                   |
| `SCHEDULER_DOWNLOADPATH` | Download path for workers                             | /data/current         |
| `SCHEDULER_UPLOADPATH`   | Upload path for workers                               | /data/processed       |
| `SCHEDULER_MINFILESIZE`  | Minimum file size for worker processing               | 100000000             |
| `WEB_PORT`               | Web server port                                       | 8080                  |
| `WEB_TOKEN`              | Web server token                                      | admin                 |

#### Worker

| Variable                   | Description                                                      | Default Value              |
| -------------------------- | ---------------------------------------------------------------- | -------------------------- |
| `BROKER_HOST`              | Broker host address                                              | localhost                  |
| `BROKER_PORT`              | Broker port                                                      | 5672                       |
| `BROKER_USER`              | Broker username                                                  | broker                     |
| `BROKER_PASSWORD`          | Broker password                                                  | broker                     |
| `BROKER_TASKENCODEQUEUE`   | Broker tasks queue name for encoding                             | tasks                      |
| `BROKER_TASKPGSQUEUE`      | Broker tasks queue name for PGS to SRT conversion                | tasks_pgstosrt             |
| `BROKER_EVENTQUEUE`        | Broker tasks events queue name                                   | task_events                |
| `LOG_LEVEL`                | Set the log level (options: "debug", "info", "warning", "error") | info                       |
| `WORKER_TEMPORALPATH`      | Path used for temporal data                                      | system temporary directory |
| `WORKER_NAME`              | Worker name used for statistics                                  | hostname                   |
| `WORKER_THREADS`           | Number of worker threads                                         | number of CPU cores        |
| `WORKER_ACCEPTEDJOBS`      | Type of jobs the worker will accept                              | ["encode"]                 |
| `WORKER_MAXPREFETCHJOBS`   | Maximum number of jobs to prefetch                               | 1                          |
| `WORKER_ENCODEJOBS`        | Number of parallel worker jobs for encoding                      | 1                          |
| `WORKER_PGJOBS`            | Number of parallel worker jobs for PGS to SRT conversion         | 0                          |
| `WORKER_PRIORITY`          | Only accept jobs of priority X                                   | 3                          |
| `WORKER_DOTNETPATH`        | Path to the dotnet executable                                    | "/usr/bin/dotnet"          |
| `WORKER_PGSTOSRTDLLPATH`   | Path to the PGSToSrt.dll library                                 | "/app/PgsToSrt.dll"        |
| `WORKER_TESSERACTDATAPATH` | Path to the tesseract data                                       | "/tessdata"                |
| `WORKER_STARTAFTER`        | Accept jobs only after the specified time (format: HH:mm)        | -                          |
| `WORKER_STOPAFTER`         | Stop accepting new jobs after the specified time (format: HH:mm) | -                          |
| `SCHEDULER_DOMAIN`         | Base domain for worker downloads and uploads                     | http://localhost:8080      |
| `SCHEDULER_SCHEDULETIME`   | Scheduling loop execution interval                               | 5m                         |
| `SCHEDULER_JOBTIMEOUT`     | Requeue jobs running for more than specified duration            | 24h                        |
| `SCHEDULER_DOWNLOADPATH`   | Download path for workers                                        | /data/current              |
| `SCHEDULER_UPLOADPATH`     | Upload path for workers                                          | /data/processed            |
| `SCHEDULER_MINFILESIZE`    | Minimum file size for worker processing                          | 100000000                  |
| `WEB_PORT`                 | Web server port                                                  | 8080                       |
| `WEB_TOKEN`                | Web server token                                                 | admin                      |

### Configuration File

The application also supports configuration through a YAML file. The default configuration file
format is YAML. If you want to use a different file format, please specify it in the `CONFIG_FILE`
environment variable.

Example YAML configuration file:

#### Server

```yaml
logLevel: info

broker:
  host: localhost
  port: 5672
  user: broker
  password: broker
  taskEncodeQueue: tasks
  taskPGSQueue: tasks_pgstosrt
  eventQueue: task_events

database:
  Driver: postgres
  Host: localhost
  port: 5432
  User: postgres
  Password: postgres
  Database: transcoder
  SSLMode: disable

scheduler:
  domain: http://localhost:8080
  scheduleTime: 5m
  jobTimeout: 24h
  downloadPath: /data/current
  uploadPath: /data/processed
  minFileSize: 100000000

web:
  port: 8080
  token: admin
```

#### Worker

```yaml
broker:
  host: localhost
  port: 5672
  user: broker
  password: broker
  taskEncodeQueue: tasks
  taskPGSQueue: tasks_pgstosrt
  eventQueue: task_events

logLevel: info

worker:
  temporalPath: /path/to/temp/data
  name: my-worker
  threads: 4
  acceptedJobs:
    - encode
  maxPrefetchJobs: 2
  encodeJobs: 2
  pgJobs: 1
  priority: 3
  dotnetPath: /usr/local/bin/dotnet
  pgsToSrtDLLPath: /custom/path/PgsToSrt.dll
  tesseractDataPath: /custom/tessdata
  startAfter: "08:00"
  stopAfter: "17:00"
```

## Client Execution

### Worker

```bash
DIR=/data/images/encode

mkdir -p $DIR
docker run -it -d --restart unless-stopped --cpuset-cpus 16-32 \
    --name transcoder-worker --hostname $(hostname) \
    -v $DIR:/tmp/ ghcr.io/pando85/transcoder:latest-worker \
    --broker.host transcoder.example.com \
    --worker.priority 9
```

**Note:** Adjust the `--cpuset-cpus` and other parameters according to your system specifications.

### PGS Worker

```bash
DIR=/data/images/pgs

mkdir -p $DIR
docker run -it -d --restart unless-stopped \
    --name transcoder-worker-pgs --hostname $(hostname) \
    -v $DIR:/tmp/ ghcr.io/pando85/transcoder:latest-worker-pgs \
    --broker.host transcoder.example.com \
    --worker.priority 9
```

**Warning:** The PGS agent must be started in advance if PGS is detected. It should run before
detection to create the RabbitMQ queue.

## Add movies from Radarr

```bash
go run ./radarr/add/main.go --api-key XXXXXX --url https://radarr.example.com --movies 5 --transcoder-url 'https://transcoder.example.com' --transcoder-token XXXXXX
```

Feel free to customize the parameters based on your Radarr and Transcoder setup.

## Update movies in Radarr

In your radarr server:

```bash
MOVIDES_DIR=/movies
find ${MOVIES_DIR} -name '*_encoded.mkv'
```

Then execute:

```
go run ./radarr/update/main.go --api-key XXXXXX --url https://radarr.example.com "${FIND_OUTPUT}"
```

Then you can go to Radarr: `Edit Movies -> Select All -> Rename Files`
