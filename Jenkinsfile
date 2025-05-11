pipeline {
  agent any

  environment {
    COMPOSE_PATH = "build/docker/miniblog"
  }

  stages {
    stage('Clean workspace') {
      steps {
        deleteDir()
      }
    }

    stage('Init') {
      steps {
        echo "✔️ Jenkins 自动完成代码 Checkout，无需手动 Clone"
        sh 'ls -lah'
      }
    }

    stage('Compose Down') {
      steps {
        dir("${COMPOSE_PATH}") {
          sh 'docker-compose down || true'
        }
      }
    }

    stage('Compose Build & Up') {
      steps {
        dir("${COMPOSE_PATH}") {
          sh 'docker-compose build'
          sh 'docker-compose up -d'
        }
      }
    }

  }

  post {
    success {
      echo '✅ 构建和部署成功'
    }
    failure {
      echo '❌ 构建或部署失败，请查看日志'
    }
  }
}
