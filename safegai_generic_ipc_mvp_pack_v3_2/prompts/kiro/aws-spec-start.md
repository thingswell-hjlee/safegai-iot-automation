SafeGAI AWS MVP Spec을 작성한다.

제품 경계:
- 현장 안전판단, 경고, Stop Request는 Ubuntu Gateway에서만 수행한다.
- AWS는 상태, 이벤트, 대표 이미지, 알림, ACK 동기화, 기본 보고서만 수행한다.
- Cloud-to-device 기계제어 API와 Topic은 금지한다.
- Gateway는 AWS 단절 중 72시간 이상 독립운전한다.

아래 세 파일을 단계적으로 작성한다.
- requirements.md: EARS 스타일 요구사항과 인수기준
- design.md: IoT Core, Lambda 2개, DynamoDB 2개, S3, Cognito, HTTP API, CloudFront, SNS, CloudWatch, CDK
- tasks.md: 1일 이하 단위, 의존관계와 자동시험 포함

먼저 requirements.md만 제시하고 승인 전 design과 implementation을 진행하지 마라.
