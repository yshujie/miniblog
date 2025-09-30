pipeline {
  agent any

  environment {
    IMAGE_REGISTRY = 'miniblog'

    BACKEND_IMAGE_TAG = "${IMAGE_REGISTRY}-backend:prod"
    FRONTEND_BLOG_IMAGE_TAG = "${IMAGE_REGISTRY}-frontend-blog:prod"
    FRONTEND_ADMIN_IMAGE_TAG = "${IMAGE_REGISTRY}-frontend-admin:prod"

    CREDENTIALS_ID = 'miniblog-dev-env'
  }

  stages {
    stage('Load Env') {
      steps {
        script {
          if (env.CREDENTIALS_ID) {
            withCredentials([file(credentialsId: env.CREDENTIALS_ID, variable: 'ENV_FILE')]) {
              def envVars = readFile(ENV_FILE).split('\n')
              envVars.each { line ->
                if (line && line.contains('=')) {
                  def (key, value) = line.split('=', 2).collect { it.trim().replaceAll(/^\"|\"$/, '') }
                  env."${key}" = value
                }
              }
            }
          }
          echo "Loaded environment variables (sensitive values hidden)"
        }
      }
    }

    stage('Checkout') {
      steps {
        deleteDir()
        checkout scm
      }
    }

    stage('Build: Frontend') {
      steps {
        dir('web/miniblog-web') {
          echo 'Building blog frontend...'
          sh 'npm ci'
          sh 'npm run build'
        }
        dir('web/miniblog-web-admin') {
          echo 'Building admin frontend...'
          sh 'npm ci'
          sh 'npm run build:prod || npm run build'
        }
        script {
          sh "docker build -f build/docker/miniblog/Dockerfile.prod.frontend.blog -t ${FRONTEND_BLOG_IMAGE_TAG} web/miniblog-web"
          sh "docker build -f build/docker/miniblog/Dockerfile.prod.frontend.admin -t ${FRONTEND_ADMIN_IMAGE_TAG} web/miniblog-web-admin"
        }
      }
    }

    stage('Build: Backend') {
      steps {
        dir('.') {
          echo 'Building backend...'
          sh 'go mod download'
          sh "docker build -f build/docker/miniblog/Dockerfile.prod.backend -t ${BACKEND_IMAGE_TAG} ."
        }
      }
    }

    stage('Prepare Network') {
      steps {
        script {
          echo 'Ensuring shared Docker network exists'
          sh '''
            if ! docker network ls --format '{{.Name}}' | grep -w infra_shared >/dev/null 2>&1; then
              docker network create infra_shared
            fi
          '''
        }
      }
    }

    stage('DB Init') {
      steps {
        script {
          if (env.SKIP_DB_INIT == 'true') {
            echo 'Skipping DB initialization (SKIP_DB_INIT=true)'
          } else {
            echo 'Running DB initialization via Makefile target db-init with Jenkins credentials'
            // 使用 Jenkins 凭据管理注入 ROOT 密码（假设凭据 ID 为 mysql-root-password）
            withCredentials([string(credentialsId: 'mysql-root-password', variable: 'DB_ROOT_PASSWORD')]) {
              sh "DB_ROOT_PASSWORD=\"${DB_ROOT_PASSWORD}\" make db-init"
            }
          }
        }
      }
    }

    stage('DB Migrate') {
      steps {
        script {
          if (env.SKIP_DB_MIGRATE == 'true') {
            echo 'Skipping DB migrations (SKIP_DB_MIGRATE=true)'
          } else {
            echo 'Running DB migrations via Makefile target db-migrate'
            def dbHost = env.MYSQL_HOST ?: 'infra-mysql'
            def dbPort = env.MYSQL_PORT ?: '3306'
            def dbUser = env.MYSQL_USERNAME ?: 'miniblog'
            def dbPassword = env.MYSQL_PASSWORD ?: 'miniblog_password'
            def dbName = env.MYSQL_DATABASE ?: 'miniblog'
            withEnv([
              "DB_HOST=${dbHost}",
              "DB_PORT=${dbPort}",
              "DB_USER=${dbUser}",
              "DB_PASSWORD=${dbPassword}",
              "DB_NAME=${dbName}"
            ]) {
              sh 'make db-migrate'
            }
          }
        }
      }
    }

    stage('Push (optional)') {
      when {
        expression { return env.PUSH_IMAGES == 'true' }
      }
      steps {
        script {
          echo 'Pushing images to registry...'
          sh "docker push ${FRONTEND_BLOG_IMAGE_TAG} || true"
          sh "docker push ${FRONTEND_ADMIN_IMAGE_TAG} || true"
          sh "docker push ${BACKEND_IMAGE_TAG} || true"
        }
      }
    }

    stage('Deploy') {
      steps {
        dir('.') {
          echo 'Deploying application using root docker-compose.yml'
          script {
            if (env.PUSH_IMAGES == 'true') {
              sh 'docker compose pull --ignore-pull-failures'
            } else {
              echo 'Skipping docker compose pull (PUSH_IMAGES!=true)'
            }
          }
          sh 'docker compose up -d'
        }
      }
    }

    stage('Cleanup') {
      steps {
        sh 'docker image prune -f || true'
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
