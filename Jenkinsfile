pipeline {
  agent any

  environment {
    // 相对项目根的 docker-compose 目录
    COMPOSE_PATH   = "build/docker/miniblog"
    // 本地 Clash 代理
    HTTP_PROXY     = "http://127.0.0.1:7890"
    HTTPS_PROXY    = "http://127.0.0.1:7890"
    // 镜像打标签
    BACKEND_IMAGE  = "yshujie/miniblog:prod"
  }

  stages {

    stage('Init') {
      steps {
        echo "✔️ 代码已经由 Jenkins 自动拉取，无需手动 git clone"
        sh 'ls -R .'
      }
    }

    stage('Build Backend Image') {
      steps {
        echo "📦 构建后端生产镜像 ${BACKEND_IMAGE}"
        // 在项目根目录执行 docker build
        sh """
          docker build --network host \
            --build-arg HTTP_PROXY=${HTTP_PROXY} \
            --build-arg HTTPS_PROXY=${HTTPS_PROXY} \
            -f build/docker/miniblog/Dockerfile.prod \
            -t ${BACKEND_IMAGE} \
            .
        """
      }
    }

    stage('Compose Down') {
      steps {
        echo "⬇️ 停止并移除旧容器（如果在运行）"
        dir("${COMPOSE_PATH}") {
          sh 'docker-compose down || true'
        }
      }
    }

    stage('Compose Build & Up') {
      steps {
        echo "🔧 重新构建并启动所有服务"
        dir("${COMPOSE_PATH}") {
          sh 'docker-compose build'
          sh 'docker-compose up -d'
        }
      }
    }

  }

  post {
    success {
      echo '✅ 全量构建与部署完成！'
    }
    failure {
      echo '❌ 构建或部署失败，请检查日志并修复'
    }
  }
}
