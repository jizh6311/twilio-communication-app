node('docker') {
  timestamps {
    properties([disableConcurrentBuilds()])

    docker.withRegistry('https://quay.io', 'quay-infrajenkins-robot-creds') {
      stage('Build') {
        checkout scm
        sh "make image"
      }

      stage('Push') {
        sh "make push-image"
      }
    }
  }
}
