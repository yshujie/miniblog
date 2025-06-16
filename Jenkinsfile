pipeline {
  agent any

  // 参数
  parameters {
    choice(name: 'ENV', choices: ['dev', 'prod'], description: '选择部署环境')
  }

  // 环境变量
  environment {
    // 项目根目录下 build/docker/miniblog
    BASE_DIR      = "build/docker/miniblog"

    // 镜像前缀
    IMAGE_REGISTRY     = 'miniblog'
    // 基础设施镜像
    NGINX_IMAGE  = "${IMAGE_REGISTRY}-nginx:prod"
    MYSQL_IMAGE  = "${IMAGE_REGISTRY}-mysql:prod"
    REDIS_IMAGE  = "${IMAGE_REGISTRY}-redis:prod"
    APP_IMAGE    = "${IMAGE_REGISTRY}-app:prod"

    // 应用镜像
    BACKEND_IMAGE_TAG  = "${IMAGE_REGISTRY}-backend:prod"
    FRONTEND_BLOG_IMAGE_TAG = "${IMAGE_REGISTRY}-frontend-blog:prod"
    FRONTEND_ADMIN_IMAGE_TAG = "${IMAGE_REGISTRY}-frontend-admin:prod"    
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

    // 加载环境变量
    stage('Load Environment') {
      steps {
        // 根据环境选择对应的凭证
        script {
          echo "🔧 加载环境变量"

          def envFile = ''
          switch(params.ENV) {
            case 'dev':
              envFile = credentials('miniblog-dev-env')
              break
            case 'prod':
              envFile = credentials('miniblog-prod-env')
              break
          }

          echo "🔧 复制环境变量文件"
          sh "cp ${envFile} .env"

          echo "🔧 设置环境变量"
          sh "export $(cat .env | xargs)"

          echo "🔧 检查环境变量"
          sh "env"
        }
      }
    }
    
    // 设置 SSL 证书，由 Jenkins 管理，写到 configs/nginx/ssl 目录下
    stage('Setup SSL') {
      steps {
        dir("${env.WORKSPACE}") {
          echo '🔧 设置 SSL 证书'

          // 从全局凭据中拉出 Secret File
          withCredentials([
            file(credentialsId: 'www.yangshujie.com.cert.key',  variable: 'WWW_SSL_KEY_FILE'),
            file(credentialsId: 'www.yangshujie.com.cert.pem',  variable: 'WWW_SSL_CRT_FILE'),
            file(credentialsId: 'admin.yangshujie.com.cert.pem',  variable: 'ADMIN_SSL_CRT_FILE'),
            file(credentialsId: 'admin.yangshujie.com.cert.key',  variable: 'ADMIN_SSL_KEY_FILE'),
            file(credentialsId: 'api.yangshujie.com.cert.key',  variable: 'API_SSL_KEY_FILE'),
            file(credentialsId: 'api.yangshujie.com.cert.pem',  variable: 'API_SSL_CRT_FILE'),
          ]) {
            sh '''
              # 创建 SSL 目录
              mkdir -p configs/nginx/ssl
              
              # 复制 www.yangshujie.com 证书
              cp "$WWW_SSL_CRT_FILE" configs/nginx/ssl/www.yangshujie.com.crt
              cp "$WWW_SSL_KEY_FILE" configs/nginx/ssl/www.yangshujie.com.key
              
              # 复制 admin.yangshujie.com 证书
              cp "$ADMIN_SSL_CRT_FILE" configs/nginx/ssl/admin.yangshujie.com.crt
              cp "$ADMIN_SSL_KEY_FILE" configs/nginx/ssl/admin.yangshujie.com.key

              # 复制 api.yangshujie.com 证书
              cp "$API_SSL_CRT_FILE" configs/nginx/ssl/api.yangshujie.com.crt
              cp "$API_SSL_KEY_FILE" configs/nginx/ssl/api.yangshujie.com.key
              
              # 设置权限
              chmod 644 configs/nginx/ssl/*.crt
              chmod 600 configs/nginx/ssl/*.key
            '''
          }
        }
      }
    }

    // 构建基础设施镜像
    stage('Infra: build') {
      steps {
        dir("${env.WORKSPACE}") {
          echo '🔧 构建基础设施镜像'

          sh "docker buildx build --no-cache -f ${BASE_DIR}/Dockerfile.infra.mysql -t ${MYSQL_IMAGE} ."
          sh "docker buildx build --no-cache -f ${BASE_DIR}/Dockerfile.infra.redis -t ${REDIS_IMAGE} ."

          // 查看镜像
          sh "docker images | grep ${IMAGE_REGISTRY}"
        }
      }
    }

    // 拉取基础设施镜像并启动基础设施容器
    stage('Infra: Up') {
      steps {
        dir("${BASE_DIR}") {
          echo '🔧 拉取基础设施镜像'

          // 启动基础设施容器
          sh 'docker compose -f compose-prod-infra.yml up -d --remove-orphans --force-recreate'

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

    // 构建前端生产镜像
    stage('Build: Frontend') {
      steps {
        dir("${BASE_DIR}") {
          echo '📦 构建前端生产镜像'

          // 构建博客前端生产镜像
          echo '📦 构建博客前端生产镜像'
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

          // 构建管理后台前端生产镜像
          echo '📦 构建管理后台前端生产镜像'
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

    // 构建后端生产镜像
    stage('Build: Backend') {
      steps {
        dir("${BASE_DIR}") {
          echo '📦 构建后端生产镜像'
          sh '''
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
            '''
        }
      }
    }

    // 构建 Nginx 镜像
    stage('Build: Nginx') {
      steps {
        dir("${env.WORKSPACE}") {
          echo '📦 构建 Nginx 生产镜像'
          sh "docker buildx build --no-cache -f ${BASE_DIR}/Dockerfile.infra.nginx -t ${NGINX_IMAGE} ."

          sh "docker images | grep ${IMAGE_REGISTRY}"
        }
      }
    }
   
    // 部署应用
    stage('App Deploy') {
      steps {
        dir("${BASE_DIR}") {
          echo '🚀 部署应用'
          sh '''
            docker compose -f compose-prod-app.yml up -d
          '''
          // 检查 Nginx 服务
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

    // 清理构建缓存
    stage('Cleanup') {
      steps {
        dir("${BASE_DIR}") { 
          echo '🧹 清理构建缓存'
          sh 'docker system prune -f'
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
    always {
      // 清理敏感文件
      sh 'rm -f .env'
    }
  }
}
