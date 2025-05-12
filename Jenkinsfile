pipeline {
  agent any

  environment {
    // 项目根目录下 build/docker/miniblog
    BASE_DIR      = "build/docker/miniblog"
    // Docker Hub 凭据 ID
    DOCKER_CREDENTIALS = 'docker-hub-credentials'
    // 镜像前缀
    IMAGE_REGISTRY     = 'yshujie'
    BACKEND_IMAGE_TAG  = "${IMAGE_REGISTRY}/miniblog:prod"
    FRONTEND_IMAGE_TAG = "${IMAGE_REGISTRY}/miniblog-frontend:prod"

    // 证书文件
    SSL_CERT = credentials('ssl-cert')
    SSL_KEY = credentials('ssl-key')
  }

  stages {
    stage('Setup SSL') {
      steps {
        dir("${BASE_DIR}") {
          // 创建证书目录
          sh 'mkdir -p configs/nginx/ssl'
          
          // 写入证书文件
          writeFile file: 'configs/nginx/ssl/yangshujie.com.crt', text: "${SSL_CERT}"
          writeFile file: 'configs/nginx/ssl/yangshujie.com.key', text: "${SSL_KEY}"
          
          // 设置正确的权限
          sh '''
            chmod 600 configs/nginx/ssl/yangshujie.com.key
            chmod 644 configs/nginx/ssl/yangshujie.com.crt
          '''
        }
      }
    }

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
          
          // 等待 MySQL 就绪
          sh '''
            until docker exec miniblog-mysql-1 mysqladmin ping -h localhost --silent; do
              echo "Waiting for MySQL..."
              sleep 2
            done
          '''
          
          // 等待 Redis 就绪
          sh '''
            until docker exec miniblog-redis-1 redis-cli ping; do
              echo "Waiting for Redis..."
              sleep 2
            done
          '''
        }
      }
    }

    // 构建后端生产镜像
    stage('Build: Backend') {
      steps {
        dir("${BASE_DIR}") {
          echo '📦 构建后端生产镜像'
          // 关闭 BuildKit，构建后端服务
          withEnv(["DOCKER_BUILDKIT=0"]) {
            sh '''
              docker build \
                --network host \
                --add-host=host.docker.internal:host-gateway \
                --build-arg GOPROXY=https://goproxy.cn,direct \
                --build-arg HTTP_PROXY=http://host.docker.internal:7890 \
                --build-arg HTTPS_PROXY=http://host.docker.internal:7890 \
                -f Dockerfile.prod.backend \
                -t ${BACKEND_IMAGE_TAG} \
                ../../../
            '''
          }
        }
      }
    }

    // 构建前端生产镜像
    stage('Build: Frontend') {
      steps {
        dir("${BASE_DIR}") {
          echo '📦 构建前端生产镜像'
          sh """
              docker build \
                --network host \
                --add-host host.docker.internal:host-gateway \
                --build-arg HTTP_PROXY=http://host.docker.internal:7890 \
                --build-arg HTTPS_PROXY=http://host.docker.internal:7890 \
                -f Dockerfile.prod.frontend \
                -t ${FRONTEND_IMAGE_TAG} \
                ../../../web/miniblog-web
              """
        }
      }
    }

    stage('App Deploy') {
      steps {
        dir("${BASE_DIR}") {
          echo '🚀 部署应用'
          sh '''
            docker-compose -f compose-prod-app.yml up -d
          '''
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
