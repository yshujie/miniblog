// Jenkinsfile
pipeline {
  agent any
  environment {
    COMPOSE_INFRA = "build/docker/miniblog/infra-compose.yml"
    COMPOSE_APP   = "build/docker/miniblog/app-compose.yml"
  }

  stages {
    stage('✅ Infra Setup') {
      steps {
        dir('build/docker/miniblog') {
          sh 'docker-compose -f ${COMPOSE_INFRA} pull || true'
          sh 'docker-compose -f ${COMPOSE_INFRA} up -d'
        }
      }
    }

    stage('🚀 Build & Deploy App') {
      steps {
        dir('build/docker/miniblog') {
          // 停掉旧的后端+nginx（不 touch 数据卷）
          sh 'docker-compose -f ${COMPOSE_APP} down || true'
          // 分别重建后端和前端打包
          sh 'docker-compose -f ${COMPOSE_APP} build backend frontend-build'
          // 运行后端+nginx（frontend-build 只是一次性容器，不需要 up）
          sh 'docker-compose -f ${COMPOSE_APP} up -d backend nginx'
        }
      }
    }
  }

  post {
    success { echo '🎉 全部服务部署成功' }
    failure { echo '❌ 部署失败，请检查日志' }
  }
}
