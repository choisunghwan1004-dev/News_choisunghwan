# News_choisunghwan GitHub 업로드 안내

연결된 GitHub 계정: `choisunghwan1004-dev`
저장소 이름: `News_choisunghwan`
권장 공개 범위: **Private** (원본 교육 자료의 재배포 권한 확인 전까지)

## 방법 A — GitHub CLI로 생성하고 업로드

Windows에 Git과 GitHub CLI를 설치하고, CMD에서 `gh auth login`으로 로그인합니다.
이 폴더에서 `CREATE_REPO_AND_PUBLISH.bat`를 실행하세요.
자동으로 Private 저장소 `choisunghwan1004-dev/News_choisunghwan`를 만들고 파일을 push합니다.
기존 저장소가 있으면 만들지 않고 중지합니다.

## 방법 B — 웹에서 저장소 생성 후 업로드

1. https://github.com/new 로 이동해 **Repository name**에 `News_choisunghwan` 입력
2. **Private** 선택, `Add a README file` 등 초기화 옵션은 비운 상태로 `Create repository` 클릭
3. Windows에 Git 설치 및 인증 후, 이 폴더에서 `PUSH_EXISTING_REPO.bat` 실행
4. https://github.com/choisunghwan1004-dev/News_choisunghwan 에서 `src/`, `demo/`, `samples/`, `original/` 등 확인

## 저장소 업로드 전 확인

- 개인 API 키, 토큰, 비밀번호를 추가하지 마세요. `.gitignore`를 함께 배포했습니다.
- Jupyter 노트북 및 `original/` 자료의 저작권과 공개 가능 범위는 별도로 확인하세요.
- EXE 2개는 편의를 위해 포함했습니다. 최신 수정 시 소스와 실행파일의 버전을 함께 맞추세요.
- Private 저장소는 GitHub에 로그인한 사용자 및 권한을 부여한 사람만 볼 수 있습니다.

## 기술적 제한

이 대화의 GitHub 연결에서는 기존 저장소 읽기·파일 쓰기가 가능하지만 새 저장소를 생성하는 기능이 제공되지 않았습니다. 따라서 이 ZIP 자체가 GitHub에 업로드되었다고 표시해서는 안 됩니다.
