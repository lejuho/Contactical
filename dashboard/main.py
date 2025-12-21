from fastapi import FastAPI
from fastapi.middleware.cors import CORSMiddleware
import requests
import random
import json

app = FastAPI()

# 프론트엔드(React)에서 접속 허용
app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_methods=["*"],
    allow_headers=["*"],
)

# 블록체인 REST API 주소 (Ignite 기본값)
NODE_API_URL = "http://localhost:1317"

@app.get("/claims")
def get_claims():
    # 블록체인 노드(1317)에서 데이터 가져오기
    response = requests.get(f"{NODE_API_URL}/contactical/reality/v1/claim")
    data = response.json()
    
    claims = data.get("claim", [])
    features = []

    for claim in claims:
        # [수정] 블록체인에 저장된 int64 좌표를 1,000,000으로 나누어 복원
        try:
            raw_lat = float(claim.get("latitude", 37566500))
            raw_lng = float(claim.get("longitude", 126978000))

            # 값이 정수형 포맷(큰 수)일 경우에만 나누기 수행
            # 위도 범위(-90~90)를 고려하여 절대값이 1000보다 크면 나눗셈 수행
            if abs(raw_lat) > 1000:
                lat = raw_lat / 1000000.0
            else:
                lat = raw_lat

            if abs(raw_lng) > 1000:
                lng = raw_lng / 1000000.0
            else:
                lng = raw_lng

        except (ValueError, TypeError):
            lat, lng = 37.5665, 126.9780 # 실패 시 기본값

        is_emergency = "#SOS" in claim.get("payload", "") or claim.get("reward_multiplier", 1) > 1
        
        feature = {
            "type": "Feature",
            "geometry": {
                "type": "Point",
                "coordinates": [lng, lat] # GeoJSON은 [경도, 위도] 순서
            },
            "properties": {
                "creator": claim.get("creator"),
                "score": int(claim.get("trust_score", 0)),
                "is_emergency": is_emergency,
                "reward": int(claim.get("trust_score", 0)) * int(claim.get("reward_multiplier", 1)) * 1000
            }
        }
        features.append(feature)

    return {"type": "FeatureCollection", "features": features}

# 실행: uvicorn main:app --reload --port 8000
