pipeline {
  agent any

  //   // Shared defaults consumed during initialisation
  environment {
    DEFAULT_IMAGE_NAMESPACE = 'miniblog'
    DEFAULT_IMAGE_TAG = 'prod'
    DOCKER_NETWORK = 'miniblog_net'
    // 强制执行 DB Init（首次部署需要）- 数据库已手动创建，现在注释掉
    // FORCE_DB_INIT = 'true'
  } pipeline hygiene options
  options {
    skipDefaultCheckout(true)
    timestamps()
    disableConcurrentBuilds()
    buildDiscarder(logRotator(numToKeepStr: '10'))
  }

  // Runtime toggles exposed to Jenkins UI users
  parameters {
    string(name: 'IMAGE_REGISTRY', defaultValue: 'miniblog', description: 'Docker registry namespace or repository prefix used for image tags.')
    string(name: 'IMAGE_TAG', defaultValue: 'prod', description: 'Tag suffix applied to built Docker images.')
    booleanParam(name: 'LOAD_ENV_FROM_CREDENTIALS', defaultValue: true, description: 'Load .env style variables from the provided Jenkins credentials file.')
    string(name: 'ENV_CREDENTIALS_ID', defaultValue: 'miniblog-dev-env', description: 'Credentials ID containing the .env file used to populate environment variables.')
    booleanParam(name: 'RUN_TESTS', defaultValue: true, description: 'Run backend unit tests before building images.')
    booleanParam(name: 'SKIP_FRONTEND_BUILD', defaultValue: false, description: 'Skip building frontend assets and Docker images.')
    booleanParam(name: 'SKIP_BACKEND_BUILD', defaultValue: false, description: 'Skip building the backend Docker image.')
    booleanParam(name: 'SKIP_DB_INIT', defaultValue: false, description: 'Skip the database initialisation stage (recommended for established environments).')
    booleanParam(name: 'SKIP_DB_MIGRATE', defaultValue: false, description: 'Skip executing database migrations.')
    booleanParam(name: 'PUSH_IMAGES', defaultValue: false, description: 'Push built Docker images to the registry.')
    booleanParam(name: 'DEPLOY_AFTER_BUILD', defaultValue: true, description: 'Deploy the stack with docker compose after build.')
    string(name: 'DEPLOY_COMPOSE_FILES', defaultValue: 'docker-compose.yml docker-compose.prod.yml', description: 'Space separated list of docker compose files used for deployment.')
    booleanParam(name: 'PRUNE_IMAGES', defaultValue: false, description: 'Prune dangling Docker images after the pipeline finishes.')
    string(name: 'DB_ROOT_CREDENTIALS_ID', defaultValue: 'mysql-root-password', description: 'Credentials ID that stores the MySQL root password for the db-init stage.')
  }

  // Shared defaults consumed during initialisation
  environment {
    DEFAULT_IMAGE_NAMESPACE = 'miniblog'
    DEFAULT_IMAGE_TAG = 'prod'
    DOCKER_NETWORK = 'miniblog_net'
  }

  stages {
    stage('Checkout') {
      steps {
        deleteDir()
        checkout scm
      }
    }

    stage('Setup') {
      steps {
        script {
          // Merge parameter and credentials based configuration
          initializeEnvironment()
        }
      }
    }

    stage('Unit Tests') {
      when {
        expression { env.RUN_TESTS == 'true' }
      }
      steps {
        dir('.') {
          echo 'Running backend unit tests...'
          sh 'make test-backend'
        }
      }
    }

    // Build both frontends in parallel when needed
    stage('Build Frontend Images') {
      when {
        expression { env.RUN_FRONTEND_BUILD == 'true' }
      }
      parallel {
        stage('Blog Frontend') {
          steps {
            dir('.') {
              sh "IMAGE_NAME='${env.FRONTEND_BLOG_IMAGE_TAG}' make docker-build-frontend-blog"
            }
          }
        }
        stage('Admin Frontend') {
          steps {
            dir('.') {
              sh "IMAGE_NAME='${env.FRONTEND_ADMIN_IMAGE_TAG}' make docker-build-frontend-admin"
            }
          }
        }
      }
    }

    stage('Build Backend Image') {
      when {
        expression { env.RUN_BACKEND_BUILD == 'true' }
      }
      steps {
        dir('.') {
          sh "IMAGE_NAME='${env.BACKEND_IMAGE_TAG}' make docker-build-backend"
        }
      }
    }

    stage('Prepare Network') {
      steps {
        dir('.') {
          sh "NETWORK='${env.DOCKER_NETWORK}' make docker-network-ensure"
        }
      }
    }

    stage('Push Images') {
      when {
        expression { env.PUSH_IMAGES_FLAG == 'true' }
      }
      steps {
        dir('.') {
          sh 'scripts/push-images.sh'
        }
      }
    }

    stage('Deploy') {
      when {
        expression { env.RUN_DEPLOY == 'true' }
      }
      steps {
        dir('.') {
          withEnv([
            "DEPLOY_COMPOSE_FILES=${env.DEPLOY_COMPOSE_FILES}",
            "PULL_IMAGES=${env.PUSH_IMAGES_FLAG}"
          ]) {
            sh 'scripts/deploy.sh'
          }
        }
      }
    }

    stage('DB Init') {
      when {
        expression { env.RUN_DB_INIT == 'true' }
      }
      steps {
        withCredentials([string(credentialsId: params.DB_ROOT_CREDENTIALS_ID, variable: 'DB_ROOT_PASSWORD')]) {
          dir('.') {
            sh 'scripts/db-init.sh'
          }
        }
      }
    }

    stage('DB Migrate') {
      when {
        expression { env.RUN_DB_MIGRATE == 'true' }
      }
      steps {
        dir('.') {
          sh 'scripts/db-migrate.sh'
        }
      }
    }

    stage('Cleanup') {
      when {
        expression { env.RUN_IMAGE_PRUNE == 'true' }
      }
      steps {
        dir('.') {
          sh 'make docker-prune-images'
        }
      }
    }
  }

  post {
    success {
      echo '✅ Pipeline succeeded.'
    }
    failure {
      echo '❌ Pipeline failed.'
    }
  }
}

def initializeEnvironment() {
  if (flagEnabled(params.LOAD_ENV_FROM_CREDENTIALS) && params.ENV_CREDENTIALS_ID?.trim()) {
    loadEnvFromCredentials(params.ENV_CREDENTIALS_ID.trim())
  } else {
    echo 'Skipping credentials based environment load.'
  }

  env.IMAGE_REGISTRY = normalizeRegistry(params.IMAGE_REGISTRY) ?: env.DEFAULT_IMAGE_NAMESPACE
  env.IMAGE_TAG = params.IMAGE_TAG?.trim() ? params.IMAGE_TAG.trim() : env.DEFAULT_IMAGE_TAG

  env.BACKEND_IMAGE_TAG = buildImageTag(env.IMAGE_REGISTRY, 'backend', env.IMAGE_TAG)
  env.FRONTEND_BLOG_IMAGE_TAG = buildImageTag(env.IMAGE_REGISTRY, 'frontend-blog', env.IMAGE_TAG)
  env.FRONTEND_ADMIN_IMAGE_TAG = buildImageTag(env.IMAGE_REGISTRY, 'frontend-admin', env.IMAGE_TAG)

  env.RUN_TESTS = flagEnabled(params.RUN_TESTS) ? 'true' : 'false'
  env.RUN_FRONTEND_BUILD = shouldSkip(params.SKIP_FRONTEND_BUILD, env.SKIP_FRONTEND_BUILD) ? 'false' : 'true'
  env.RUN_BACKEND_BUILD = shouldSkip(params.SKIP_BACKEND_BUILD, env.SKIP_BACKEND_BUILD) ? 'false' : 'true'
  // 检查是否有 FORCE_DB_INIT 环境变量强制执行
  env.RUN_DB_INIT = flagEnabled(env.FORCE_DB_INIT) ? 'true' : (shouldSkip(params.SKIP_DB_INIT, env.SKIP_DB_INIT) ? 'false' : 'true')
  env.RUN_DB_MIGRATE = shouldSkip(params.SKIP_DB_MIGRATE, env.SKIP_DB_MIGRATE) ? 'false' : 'true'

  def pushImages = flagEnabled(params.PUSH_IMAGES)
  if (!pushImages) {
    pushImages = flagEnabled(env.PUSH_IMAGES)
  }
  env.PUSH_IMAGES_FLAG = pushImages ? 'true' : 'false'

  def deployAfterBuild = flagEnabled(params.DEPLOY_AFTER_BUILD)
  if (!deployAfterBuild) {
    deployAfterBuild = flagEnabled(env.DEPLOY_AFTER_BUILD)
  }
  env.RUN_DEPLOY = deployAfterBuild && !flagEnabled(env.SKIP_DEPLOY) ? 'true' : 'false'

  env.DEPLOY_COMPOSE_FILES = params.DEPLOY_COMPOSE_FILES?.trim() ? params.DEPLOY_COMPOSE_FILES.trim() : 'docker-compose.yml docker-compose.prod.yml'

  def pruneImages = flagEnabled(params.PRUNE_IMAGES)
  if (!pruneImages) {
    pruneImages = flagEnabled(env.PRUNE_IMAGES)
  }
  env.RUN_IMAGE_PRUNE = pruneImages ? 'true' : 'false'

  echo "Using Docker image namespace: ${env.IMAGE_REGISTRY}"
  echo "Using Docker image tag: ${env.IMAGE_TAG}"
  echo "Backend image tag: ${env.BACKEND_IMAGE_TAG}"
  echo "Blog frontend image tag: ${env.FRONTEND_BLOG_IMAGE_TAG}"
  echo "Admin frontend image tag: ${env.FRONTEND_ADMIN_IMAGE_TAG}"
  echo "Run unit tests: ${env.RUN_TESTS}"
  echo "Run frontend build: ${env.RUN_FRONTEND_BUILD}"
  echo "Run backend build: ${env.RUN_BACKEND_BUILD}"
  echo "Run db init: ${env.RUN_DB_INIT}"
  echo "Run db migrate: ${env.RUN_DB_MIGRATE}"
  echo "Push images: ${env.PUSH_IMAGES_FLAG}"
  echo "Deploy after build: ${env.RUN_DEPLOY}"
  echo "Deployment compose files: ${env.DEPLOY_COMPOSE_FILES}"
  echo "Prune images after pipeline: ${env.RUN_IMAGE_PRUNE}"
}

def loadEnvFromCredentials(String credentialsId) {
  withCredentials([file(credentialsId: credentialsId, variable: 'ENV_FILE')]) {
    def content = readFile(ENV_FILE)
    def target = "${env.WORKSPACE}/.pipeline.env"
    writeFile file: target, text: content
    env.PIPELINE_ENV_FILE = target

    def exposedKeys = content.split('\n')
      .findAll { line ->
        def trimmed = line.trim()
        trimmed && !trimmed.startsWith('#') && trimmed.contains('=')
      }
      .collect { entry ->
        entry.split('=', 2)[0].replaceFirst(/^export\s+/, '').trim()
      }

    echo "Loaded environment file from credentials '${credentialsId}' (keys: ${exposedKeys.join(', ')}; sensitive values hidden)."
  }
}

boolean flagEnabled(def value) {
  if (value == null) {
    return false
  }
  def normalized = value.toString().trim().toLowerCase()
  return ['1', 'true', 'yes', 'y'].contains(normalized)
}

boolean shouldSkip(def paramValue, def envValue) {
  return flagEnabled(paramValue) || flagEnabled(envValue)
}

String normalizeRegistry(Object input) {
  def raw = input?.toString()?.trim()
  if (!raw) {
    return ''
  }
  return raw.endsWith('/') ? raw[0..-2] : raw
}

String buildImageTag(String registry, String component, String tag) {
  def base = registry?.trim()
  def repo = base ? "${base}-${component}" : component
  return "${repo}:${tag}"
}
