pipeline {
	agent any
	tools { go 'go1.20' }
	stages {
		stage('test') {
			steps {
				sh 'go test ./...'
			}
		}
		stage('build') {
			steps {
				sh 'go build ./...'
			}
		}
	}
}
