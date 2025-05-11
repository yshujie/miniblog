pipeline {
  agent any

  environment {
    COMPOSE_PATH = "build/docker/miniblog"
  }

  stages {
    stage('Clone') {
      steps {
        git credentialsId: 'github-token', url: 'https://github.com/yshujie/miniblog.git'
      }
    }

    stage('Compose Up') {
      steps {
        dir("${COMPOSE_PATH}") {
          sh 'docker-compose down'
          sh 'docker-compose build'
          sh 'docker-compose up -d'
        }
      }
    }
  }
}
