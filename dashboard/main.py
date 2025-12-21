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
    response = requests.get(f"{NODE_API_URL}/contactical/reality/v1/claim")
    data = response.json()
    
    claims = data.get("claim", [])
    features = []

    for claim in claims:
        # 블록체인에서 온 실제 좌표 사용 (랜덤 삭제!)
        lat = float(claim.get("latitude", 37.5665))
        lng = float(claim.get("longitude", 126.9780))

    # 2. GeoJSON 포맷으로 변환 (지도 라이브러리 표준)
    for claim in claims:
        # [시뮬레이션] 실제 GPS가 없으므로 서울 근처 랜덤 좌표 생성
        lat = 37.5665 + random.uniform(-0.05, 0.05)
        lng = 126.9780 + random.uniform(-0.05, 0.05)
        
        # #SOS 여부 확인
        is_emergency = "#SOS" in claim.get("payload", "") or claim.get("reward_multiplier", 1) > 1
        
        feature = {
            "type": "Feature",
            "geometry": {
                "type": "Point",
                "coordinates": [lng, lat]
            },
            "properties": {
                "creator": claim.get("creator"),
                "score": int(claim.get("trust_score", 10)),
                "is_emergency": is_emergency,
                "reward": int(claim.get("trust_score", 10)) * int(claim.get("reward_multiplier", 1)) * 1000
            }
        }
        features.append(feature)

    return {"type": "FeatureCollection", "features": features}

# 실행: uvicorn main:app --reload --port 8000