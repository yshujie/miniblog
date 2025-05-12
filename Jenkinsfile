pipeline {
  agent any

  environment {
    // 项目根目录下 build/docker/miniblog
    BASE_DIR      = "build/docker/miniblog"
    // 代理（如需要）
    HTTP_PROXY    = "http://127.0.0.1:7890"
    HTTPS_PROXY   = "http://127.0.0.1:7890"
    // Docker Hub 凭据 ID
    DOCKER_CREDENTIALS = 'docker-hub-credentials'
    // 镜像前缀
    IMAGE_REGISTRY     = 'yshujie'
    BACKEND_IMAGE_TAG  = "${IMAGE_REGISTRY}/miniblog:prod"
    FRONTEND_IMAGE_TAG = "${IMAGE_REGISTRY}/miniblog-frontend:prod"
  }

  stages {
    stage('Checkout') {
      steps {
        // 清理旧内容
        deleteDir()
        // 拉取最新代码
        checkout scm
      }
    }

    stage('Infra: Pull & Up') {
      steps {
        dir("${BASE_DIR}") {
          // 拉取基础镜像
          sh 'docker-compose -f compose-prod-infra.yml pull'
          // 启动基础设施
          sh 'docker-compose -f compose-prod-infra.yml up -d'
          // 休眠 10 秒
          sleep 10
        }
      }
    }

    stage('Build & Push: Backend') {
      steps {
        dir("${BASE_DIR}") {
          withCredentials([usernamePassword(
            credentialsId: DOCKER_CREDENTIALS,
            usernameVariable: 'DOCKER_USER',
            passwordVariable: 'DOCKER_PASS'
          )]) {
            // 构建后端镜像
            sh """
              docker login -u "$DOCKER_USER" -p "$DOCKER_PASS"
              docker build \
                --network host \
                --build-arg HTTP_PROXY=${HTTP_PROXY} \
                --build-arg HTTPS_PROXY=${HTTPS_PROXY} \
                -f Dockerfile.prod \
                -t ${BACKEND_IMAGE_TAG} \
                ../../..
            """
            // 推送到仓库
            sh "docker push ${BACKEND_IMAGE_TAG}"
          }
        }
      }
    }

    stage('Build & Push: Frontend') {
      steps {
        dir("${BASE_DIR}") {
          withCredentials([usernamePassword(
            credentialsId: DOCKER_CREDENTIALS,
            usernameVariable: 'DOCKER_USER',
            passwordVariable: 'DOCKER_PASS'
          )]) {
            // 构建前端镜像
            sh """
              docker login -u "$DOCKER_USER" -p "$DOCKER_PASS"
              docker build \
                --network host \
                --build-arg HTTP_PROXY=${HTTP_PROXY} \
                --build-arg HTTPS_PROXY=${HTTPS_PROXY} \
                -f Dockerfile.frontend \
                -t ${FRONTEND_IMAGE_TAG} \
                ../../../web/miniblog-web
            """
            // 推送到仓库
            sh "docker push ${FRONTEND_IMAGE_TAG}"
          }
        }
      }
    }

    stage('App Deploy') {
      steps {
        dir("${BASE_DIR}") {
          // 重新拉取最新镜像并启动业务容器
          sh 'docker-compose -f compose-prod-app.yml pull'
          sh 'docker-compose -f compose-prod-app.yml up -d'
        }
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
