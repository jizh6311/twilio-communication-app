node('docker') {
  timestamps {
    properties([disableConcurrentBuilds()])

    build_tag = "${env.BRANCH_NAME}-${env.BUILD_ID}"
    base_tag = "quay.io/aspenmesh/data-science-traffic-simulation:${build_tag}"
    def gitCommit
    
    stage('Checkout') {
        // see ui to enable submodule checkout
        checkout scm
        gitCommit = sh(returnStdout: true, script: "git rev-parse HEAD").trim()
    }

#    stage('Test') {
#      sh "docker build --target=tester -f Dockerfile -t ${base_tag} ."
#    }

    stage('Build') {
      // Can't use docker.build
      // (https://issues.jenkins-ci.org/browse/JENKINS-44609)
      sh "docker build --label repo=data-models --label git-commit=${gitCommit} -f Dockerfile -t ${base_tag} ."
    }

    withCredentials([[
      $class: 'UsernamePasswordMultiBinding',
      credentialsId: 'quay-infrajenkins-robot-creds',
      usernameVariable: 'QUAY_USERNAME',
      passwordVariable: 'QUAY_PASSWORD'
    ]]) {
      stage('Push') {
        docker_config="docker-config/"
        try {
          sh """
            export DOCKER_CONFIG="${docker_config}"
            if [ -e \$DOCKER_CONFIG ]; then
              rm -r ./\$DOCKER_CONFIG
            fi
            mkdir \$DOCKER_CONFIG
            docker login -u "${QUAY_USERNAME}" -p "${QUAY_PASSWORD}" quay.io
            docker push ${base_tag}
            rm -r ./\$DOCKER_CONFIG
          """
        } catch (Exception e) {
          sh """
            export DOCKER_CONFIG="${docker_config}"
            if [ -e \$DOCKER_CONFIG ]; then
              rm -r ./\$DOCKER_CONFIG
            fi
          """
          throw e
        }
      }
    }    
  }
}
