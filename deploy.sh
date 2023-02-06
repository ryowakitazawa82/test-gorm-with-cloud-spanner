gcloud run deploy --source=. \
--service-account=$SA --region=us-central1 gorm-test \
--set-env-vars=PROJECT_ID=$PROJECT_ID,INSTANCE_NAME=test-instance,DATABASE_NAME=musics \
--cpu=2 --memory=1Gi
