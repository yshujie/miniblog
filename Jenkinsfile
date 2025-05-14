pipeline {
  agent any

  environment {
    // 项目根目录下 build/docker/miniblog
    BASE_DIR      = "build/docker/miniblog"

    // 证书文件
    SSL_CERT = credentials('ssl-cert')
    SSL_KEY = credentials('ssl-key')

    // 镜像前缀
    IMAGE_REGISTRY     = 'yshujie'
    // 基础设施镜像
    NGINX_IMAGE  = "${IMAGE_REGISTRY}/miniblog-nginx:prod"
    MYSQL_IMAGE  = "${IMAGE_REGISTRY}/miniblog-mysql:prod"
    REDIS_IMAGE  = "${IMAGE_REGISTRY}/miniblog-redis:prod"
    APP_IMAGE    = "${IMAGE_REGISTRY}/miniblog-app:prod"
    // 应用镜像
    BACKEND_IMAGE_TAG  = "${IMAGE_REGISTRY}/miniblog:prod"
    FRONTEND_IMAGE_TAG = "${IMAGE_REGISTRY}/miniblog-frontend:prod"
  }

  // 阶段
  stages {
    // 拉取最新代码
    stage('Checkout') {
      steps {
        // 清理旧内容
        deleteDir()
        // 拉取最新代码
        checkout scm
      }
    }

    // 设置 SSL 证书，由 Jenkins 管理，写到 configs/nginx/ssl 目录下
    stage('Setup SSL') {
      steps {
        dir("${BASE_DIR}") {
          echo '🔧 设置 SSL 证书'

          // 创建证书目录
          sh 'mkdir -p configs/nginx/ssl'
          
          // 写入证书文件
          writeFile file: 'configs/nginx/ssl/yangshujie.com.crt', text: "${SSL_CERT}"
          writeFile file: 'configs/nginx/ssl/yangshujie.com.key', text: "${SSL_KEY}"
          
          // 设置正确的权限
          sh '''
            chmod 600 configs/nginx/ssl/yangshujie.com.key
            chmod 644 configs/nginx/ssl/yangshujie.com.crt

            # 验证证书文件
            echo "=== 证书文件权限 ==="
            ls -l configs/nginx/ssl/
            
            echo "=== 证书文件内容 ==="
            head -n 1 configs/nginx/ssl/yangshujie.com.crt
            head -n 1 configs/nginx/ssl/yangshujie.com.key
          '''
        }
      }
    }

    // 构建基础设施镜像
    stage('Infra: build') {
      steps {
        dir("${BASE_DIR}") {
          echo '🔧 构建基础设施镜像'
          sh "docker build -f Dockerfile.infra.nginx -t ${NGINX_IMAGE} ../../../"
          sh "docker build -f Dockerfile.infra.mysql -t ${MYSQL_IMAGE} ../../../"
          sh "docker build -f Dockerfile.infra.redis -t ${REDIS_IMAGE} ../../../"

          // 查看镜像
          sh "docker images | grep ${IMAGE_REGISTRY}"
        }
      }
    }

    // 拉取基础设施镜像并启动基础设施容器
    stage('Infra: Pull & Up') {
      steps {
        dir("${BASE_DIR}") {
          echo '🔧 拉取基础设施镜像'
          // 拉取基础设施镜像
          sh 'docker-compose -f compose-prod-infra.yml pull'
          // 启动基础设施容器
          sh 'docker-compose -f compose-prod-infra.yml up -d'

          // 等待 Nginx 就绪
          sh '''
            until docker exec miniblog-nginx-1 nginx -t; do
              echo "Waiting for Nginx..."
              sleep 2
            done
          '''

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

    // 部署应用
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

    // 清理构建缓存
    stage('Cleanup') {
      steps {
        echo '🧹 清理构建缓存'
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
