import grpc
from concurrent import futures
import time
import logging
import numpy as np
import pandas as pd
import statsmodels.api as sm
# Assuming we will generate the proto python code, but for now we will implement the server logic structure.
# We need to define the proto service first. 
# Let's create a placeholder for now and I will generate the proto file next.

from factor_model import FactorModel
import analytics_pb2
import analytics_pb2_grpc

class AnalyticsEngineServicer(analytics_pb2_grpc.AnalyticsEngineServicer):
    def __init__(self):
        self.model = FactorModel()

    def CalculateRegression(self, request, context):
        try:
            # Extract data from gRPC request
            X = [item.value for item in request.independent_data]
            y = [item.value for item in request.dependent_data]
            
            if len(X) != len(y) or len(X) == 0:
                 return analytics_pb2.RegressionResponse(
                    alpha=0.0, beta=0.0, r_squared=0.0, status="error: mismatch len or empty"
                )

            # Run Logic
            result = self.model.run_regression(X, y)
            
            return analytics_pb2.RegressionResponse(
                alpha=result["alpha"],
                beta=result["beta"],
                r_squared=result["r_squared"],
                status=result["status"]
            )
        except Exception as e:
             return analytics_pb2.RegressionResponse(
                alpha=0.0, beta=0.0, r_squared=0.0, status=f"server error: {str(e)}"
            )

def serve():
    server = grpc.server(futures.ThreadPoolExecutor(max_workers=10))
    analytics_pb2_grpc.add_AnalyticsEngineServicer_to_server(AnalyticsEngineServicer(), server)
    server.add_insecure_port('[::]:50051')
    server.start()
    print("Analytics Engine started on port 50051")
    server.wait_for_termination()

if __name__ == '__main__':
    logging.basicConfig()
    serve()
