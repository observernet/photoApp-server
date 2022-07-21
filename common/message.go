package common

func GetErrorMessage(lang string, code int) string {

	var mesg string

	if lang == "K" {
		mesg = GetErrorMessageKOR(code)
	} else {
		mesg = GetErrorMessageENG(code)
	}

	return mesg
}

func GetErrorMessageKOR(code int) string {

	switch code {
		case 0: 	return "정상처리"

		case 8001:	return "이미 가입된 휴대전화번호입니다"
		case 8002:	return "존재하지 않는 휴대전화번호입니다"
		case 8003:	return "이미 등록된 이메일입니다"
		case 8004:	return "존재하지 않는 이메일입니다"
		case 8005:	return "인증 실패로 인해 인증이 제한되었습니다"
		case 8006:	return "인증 요청 정보가 존재하지 않습니다"
		case 8007:	return "인증번호가 만료되었습니다"
		case 8008:	return "올바르지 않은 인증코드입니다"
		case 8009:	return "최대 인증 횟수를 초과하였습니다"
		case 8010:	return "올바르지 않은 계정 또는 비밀번호 입니다"
		case 8011:	return "로그인 실패 횟수가 초과 되었습니다"
		case 8012:	return "정책 위반으로 계정이 정지되었습니다"
		case 8013:	return "올바르지 않은 계정 상태입니다"
		case 8014:	return "로그인 정보가 올바르지 않습니다"
		case 8015:	return "로그인 정보가 존재하지 않습니다"
		case 8016:	return "계정 정보가 존재하지 않습니다"
		case 8017:	return "해당 휴대전화번호로 등록된 이메일이 없습니다"
		case 8018:	return "해당 이메일로 등록된 휴대전화번호가 없습니다"
		case 8019:	return "이미 등록된 지갑주소입니다. 관리자에게 문의하세요"
		case 8020:	return "이미 등록된 보상정보입니다. 관리자에게 문의하세요"
		case 8021:	return "올바르지 않는 지갑주소입니다"
		case 8022:	return "동일 닉네임이 존재합니다"
		case 8023:	return "금칙어 위반입니다"

		case 8101:	return "스냅 이력이 존재합니다"
		case 8102:	return "먼저 스냅한 사용자가 존재합니다"
		case 8103:	return "라벨이 부족합니다"
		case 8104:	return "스냅키가 올바르지 않습니다"
		case 8105:	return "본인의 스냅에 라벨할 수 없습니다"
		case 8106:	return "신고된 스냅은 라벨할 수 없습니다"
		case 8107:	return "스냅의 상태가 올바르지 않습니다"
		case 8108:	return "이미 라벨한 스냅입니다"
		case 8109:	return "스냅의 최대 라벨수를 초과하였습니다"

		case 8201:	return "이체 수수료 면제 티켓이 부족합니다"
		case 8202:	return "환전 가능 금액을 초과하였습니다"
		case 8203:	return "등록 주소가 존재하지 않습니다"
		case 8204:	return "처리중인 환전 내역이 있습니다. 잠시후에 시도하세요"
		case 8205:	return "환전 가능 시간이 아닙니다 (00:20 ~ 23:50)"
		case 8206:	return "입력 주소가 등록되지 않은 주소입니다"

		case 8211:	return "출금 가능 시간이 아닙니다 (00:20 ~ 23:50)"
		case 8212:	return "처리중인 출금 내역이 있습니다. 잠시후에 시도하세요"
		case 8213:	return "출금 가능 금액을 초과하였습니다"
		case 8214:	return "개인 지갑은 출금할 수 없습니다"

		case 9001:	return "검증 오류"
		case 9002:	return "요청이 만료되었습니다"
		case 9003:	return "요청 데이타 오류"
		case 9004:	return "정의되지 않은 요청"

		case 9901:	return "시스템 오류"
		case 9902:	return "잘못된 접근입니다"
	}

	return "정의되지 않은 메세지"
}

func GetErrorMessageENG(code int) string {

	switch code {
		case 0:		return "Processed"

		case 9001:	return "Validation Error"
		case 9002:	return "Your request has expired"
		case 9003:	return "Request data error"
		case 9004:	return "undefined request"
		case 9901:	return "System Error"
	}

	return "undefined message"
}
