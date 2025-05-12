// Jenkinsfile
pipeline {
  agent any
  environment {
    COMPOSE_INFRA = "compose-infra.yml"
    COMPOSE_APP   = "docker-compose.yml"
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
          // 停掉旧的服务（不 touch 数据卷）
          sh 'docker-compose -f ${COMPOSE_APP} down || true'
          // 分别重建后端和前端
          sh 'docker-compose -f ${COMPOSE_APP} build backend frontend'
          // 运行所有服务
          sh 'docker-compose -f ${COMPOSE_APP} up -d'
        }
      }
    }
  }

  post {
    success { echo '🎉 全部服务部署成功' }
    failure { echo '❌ 部署失败，请检查日志' }
  }
}
