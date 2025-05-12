// Jenkinsfile
pipeline {
  agent any
  environment {
    COMPOSE_INFRA = "compose-infra.yml"
    COMPOSE_APP   = "compose-app.yml"
  }

  stages {
    stage('✅ Infra Setup') {
      steps {
        dir('build/docker/miniblog') {
          // 启动基础设施服务
          sh 'docker-compose -f ${COMPOSE_INFRA} pull || true'
          sh 'docker-compose -f ${COMPOSE_INFRA} up -d'
          // 等待基础设施服务就绪
          sh 'sleep 10'
        }
      }
    }

    stage('🚀 Build & Deploy App') {
      steps {
        dir('build/docker/miniblog') {
          // 停掉旧的应用服务（不 touch 数据卷）
          sh 'docker-compose -f ${COMPOSE_APP} down || true'
          // 分别重建后端和前端
          sh 'docker-compose -f ${COMPOSE_APP} build backend frontend'
          // 运行应用服务
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
