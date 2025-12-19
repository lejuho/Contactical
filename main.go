package main

import (
	"context"
	"fmt"
	"log"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	// protoc로 생성된 패키지 (경로는 본인의 프로젝트 설정에 맞게 수정)
	pb "contactical/proto/contactical/contactical/api/v1" 
)

type server struct {
	pb.UnimplementedContacticalServiceServer
}

// 1. 노드 등록 (Key Attestation 검증)
func (s *server) RegisterNode(ctx context.Context, req *pb.RegisterNodeRequest) (*pb.RegisterNodeResponse, error) {
	fmt.Printf("📩 RegisterNode 요청 수신: %s\n", req.CreatorAddress)

	// [기존 Go 검증 로직 호출]
	// VerifyAttestationChain(req.CertChain, req.Challenge) 실행
	// 여기서는 예시로 성공 처리
	success := true 

	if !success {
		return nil, status.Errorf(codes.Unauthenticated, "기기 검증 실패")
	}

	return &pb.RegisterNodeResponse{
		Success: true,
		Message: "노드 등록 성공",
		NodeId:  "node_" + req.CreatorAddress[:8],
	}, nil
}

// 2. 데이터 제출 (TEE 서명 검증)
func (s *server) SubmitData(ctx context.Context, req *pb.SubmitDataRequest) (*pb.SubmitDataResponse, error) {
	fmt.Printf("📩 SubmitData 요청 수신! NodeID: %s, Payload: %s\n", req.NodeId, req.Payload)

	// [기존 Go 서명 검증 로직 호출]
	// isValid, _ := VerifyDataSignature(req.Payload, req.Signature, req.Cert)
	isValid := true 

	if !isValid {
		return &pb.SubmitDataResponse{Success: false}, nil
	}

	return &pb.SubmitDataResponse{
		Success: true,
		TxHash:  "0xabc123...", // 실제로는 블록체인 트랜잭션 해시가 들어감
	}, nil
}

func main() {
	// gRPC 기본 포트는 보통 9090을 많이 씁니다. (8080과 겹치지 않게)
	lis, err := net.Listen("tcp", "0.0.0.0:9090")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer()
	pb.RegisterContacticalServiceServer(s, &server{})

	fmt.Println("🚀 Contactical gRPC 서버 시작 (Port 9090)...")
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}