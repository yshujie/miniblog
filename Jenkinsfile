pipeline {
  agent any

  environment {
    BASE_DIR                = "build/docker/miniblog"
    IMAGE_REGISTRY          = 'miniblog'

    NGINX_IMAGE             = "${IMAGE_REGISTRY}-nginx:prod"
    MYSQL_IMAGE             = "${IMAGE_REGISTRY}-mysql:prod"
    REDIS_IMAGE             = "${IMAGE_REGISTRY}-redis:prod"
    APP_IMAGE               = "${IMAGE_REGISTRY}-app:prod"

    BACKEND_IMAGE_TAG       = "${IMAGE_REGISTRY}-backend:prod"
    FRONTEND_BLOG_IMAGE_TAG = "${IMAGE_REGISTRY}-frontend-blog:prod"
    FRONTEND_ADMIN_IMAGE_TAG= "${IMAGE_REGISTRY}-frontend-admin:prod"

    CREDENTIALS_ID          = 'miniblog-dev-env'
  }

  stages {
    stage('Load Env') {
      steps {
        script {
          withCredentials([file(credentialsId: env.CREDENTIALS_ID, variable: 'ENV_FILE')]) {
            def envVars = readFile(ENV_FILE).split("\n")
            envVars.each { line ->
              if (line && line.contains('=')) {
                def (key, value) = line.split('=', 2).collect { it.trim().replaceAll(/^["']|["']$/, '') }
                env."${key}" = value
              }
            }
          }

          // 验证加载的变量
          echo "MYSQL_HOST: ${env.MYSQL_HOST}"
          echo "MYSQL_PORT: ${env.MYSQL_PORT}"
          echo "MYSQL_USER: ${env.MYSQL_USER}"
          echo "MYSQL_NAME: ${env.MYSQL_NAME}"
          echo "MYSQL_PASSWORD: ${env.MYSQL_PASSWORD}"
        }
      }
    }

    stage('Checkout') {
      steps {
        deleteDir()
        checkout scm
      }
    }

    stage('Infra: build') {
      steps {
        dir("${env.WORKSPACE}") {
          echo '🔧 构建基础设施镜像'

          sh """
            docker buildx build --no-cache \
              -f ${BASE_DIR}/Dockerfile.infra.mysql \
              -t ${MYSQL_IMAGE} \
              --build-arg DB_HOST=${env.MYSQL_HOST} \
              --build-arg DB_PORT=${env.MYSQL_PORT} \
              --build-arg DB_USER=${env.MYSQL_USER} \
              --build-arg DB_NAME=${env.MYSQL_NAME} \
              --build-arg DB_PASSWORD=${env.MYSQL_PASSWORD} \
              .
          """

          sh "docker buildx build --no-cache -f ${BASE_DIR}/Dockerfile.infra.redis -t ${REDIS_IMAGE} ."
        }
      }
    }

    stage('Infra: Up') {
      steps {
        dir("${BASE_DIR}") {
          echo '🔧 拉取基础设施镜像并启动容器'

          sh """
            MYSQL_HOST=${env.MYSQL_HOST} \
            MYSQL_PORT=${env.MYSQL_PORT} \
            MYSQL_USER=${env.MYSQL_USER} \
            MYSQL_NAME=${env.MYSQL_NAME} \
            MYSQL_PASSWORD=${env.MYSQL_PASSWORD} \
            docker compose -f compose-prod-infra.yml up -d --remove-orphans --force-recreate
          """

          sh '''
            until docker exec miniblog-mysql-1 mysqladmin ping -h localhost --silent; do
              echo "Waiting for MySQL..."
              sleep 2
            done

            until docker exec miniblog-redis-1 redis-cli ping; do
              echo "Waiting for Redis..."
              sleep 2
            done
          '''
        }
      }
    }

    stage('Build: Frontend') {
      steps {
        dir("${BASE_DIR}") {
          echo '📦 构建前端生产镜像'

          sh """
            docker buildx build \
              --network host \
              --add-host host.docker.internal:host-gateway \
              --build-arg HTTP_PROXY=http://host.docker.internal:7890 \
              --build-arg HTTPS_PROXY=http://host.docker.internal:7890 \
              -f Dockerfile.prod.frontend.blog \
              -t ${FRONTEND_BLOG_IMAGE_TAG} \
              ../../../web/miniblog-web
          """

          sh """
            docker buildx build \
              --network host \
              --add-host host.docker.internal:host-gateway \
              --build-arg HTTP_PROXY=http://host.docker.internal:7890 \
              --build-arg HTTPS_PROXY=http://host.docker.internal:7890 \
              -f Dockerfile.prod.frontend.admin \
              -t ${FRONTEND_ADMIN_IMAGE_TAG} \
              ../../../web/miniblog-web-admin
          """
        }
      }
    }

    stage('Build: Backend') {
      steps {
        dir("${BASE_DIR}") {
          echo '📦 构建后端生产镜像'
          sh """
            docker buildx build \
              --network host \
              --add-host host.docker.internal:host-gateway \
              --build-arg GOPROXY=https://goproxy.cn,direct \
              --build-arg HTTP_PROXY=http://host.docker.internal:7890 \
              --build-arg HTTPS_PROXY=http://host.docker.internal:7890 \
              --build-arg GO111MODULE=on \
              --cache-from ${BACKEND_IMAGE_TAG} \
              -f Dockerfile.prod.backend \
              -t ${BACKEND_IMAGE_TAG} \
              ../../../
          """
        }
      }
    }

    stage('Build: Nginx') {
      steps {
        dir("${env.WORKSPACE}") {
          echo '📦 构建 Nginx 生产镜像'
          sh "docker buildx build --no-cache -f ${BASE_DIR}/Dockerfile.infra.nginx -t ${NGINX_IMAGE} ."
        }
      }
    }

    stage('App Deploy') {
      steps {
        dir("${BASE_DIR}") {
          echo '🚀 部署应用'
          sh "docker compose -f compose-prod-app.yml up -d"

          sh '''
            until docker exec miniblog-nginx-1 nginx -t; do
              echo "Waiting for Nginx..."
              sleep 2
            done
            echo "🚀 Nginx started successfully"
          '''
        }
      }
    }

    stage('Cleanup') {
      steps {
        sh 'docker system prune -f'
      }
    }
  }

  post {
    success {
      echo '✅ 部署完成！'
    }
    failure {
      echo '❌ 部署失败，请检查日志并修复。'
    }
  }
}
