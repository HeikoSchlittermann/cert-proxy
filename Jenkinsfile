pipeline {
    agent any
    options { disableConcurrentBuilds abortPrevious: true }
    tools { go 'go1.20' }

    environment {
        pkgUrl = 'https://gitea.schlittermann.de/api/packages/ius/generic/cert-proxy'
    }

    stages {
        stage('prepare') {
            steps {
                withCredentials([gitUsernamePassword(credentialsId: 'jenkins@gitea')]) {
                    sh 'git fetch --tags --prune'
                }
            }
        }
        stage('update') {
            when { branch 'ci' }
            environment { // this we do need for the commits
                GIT_AUTHOR_EMAIL    = "jenkins@schlittermann.de"
                GIT_COMMITTER_EMAIL = "jenkins@schlittermann.de"
                GIT_AUTHOR_NAME     = "Mr. Jenkins"
                GIT_COMMITTER_NAME  = "Mr. Jenkins"
            }
            steps {
                withCredentials([gitUsernamePassword(credentialsId: 'jenkins@gitea')]) {
                    sh '''
                        git fetch origin master
                        git merge -X theirs FETCH_HEAD
                        make update
                        if git commit -am 'ci: automatic dependency update'
                        then
                            git checkout "origin/$BRANCH_NAME"
                            git merge HEAD@{1}
                            git push origin "HEAD:$BRANCH_NAME"
                        fi
                    '''
                }
            }
        }
        stage('test') {
            steps {
                sh 'make test'
            }
        }
        stage('build') {
            when { branch 'master' }
            steps {
                sh 'make clean all'
            }
        }
    }
    post {
        /* for now we do not upload anything *
        success {
            withCredentials([string(credentialsId: 'heiko@gitea', variable: 'CRED')]) {
                sh '''
                    test "$BRANCH_NAME" = master || exit 0
                    version=$(git describe --always)
                    curl -H "Authorization: token $CRED" -X DELETE "$pkgUrl/$version"
                    curl -H "Authorization: token $CRED" --upload-file "out/cert-proxy-{client,server}" "$pkgUrl/$version/"
                '''
            }
        }
        */
        cleanup {
            sh 'make clean'
        }
    }
}

// vim:tw=0 sts=4 sw=4 ai si et:
