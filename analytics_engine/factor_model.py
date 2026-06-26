# analytics_engine/factor_model.py
import numpy as np
import pandas as pd
import statsmodels.api as sm

class FactorModel:
    def __init__(self):
        pass

    def run_regression(self, independent_vars, dependent_var):
        """
        Runs an OLS regression.
        
        Args:
            independent_vars (list or np.array): The X values (Market, etc.)
            dependent_var (list or np.array): The y values (Portfolio Returns)
            
        Returns:
            dict: {alpha, beta, r_squared, status}
        """
        try:
            # Ensure numpy arrays
            X = np.array(independent_vars)
            y = np.array(dependent_var)
            
            # Add constant for Intercept (Alpha)
            X = sm.add_constant(X)
            
            # Fit Model
            model = sm.OLS(y, X).fit()
            
            # Extract Results
            # params[0] is const (Alpha), params[1] is x1 (Beta)
            alpha = float(model.params[0]) if len(model.params) > 0 else 0.0
            beta = float(model.params[1]) if len(model.params) > 1 else 0.0
            r_squared = float(model.rsquared)
            
            return {
                "alpha": alpha,
                "beta": beta,
                "r_squared": r_squared,
                "status": "success"
            }
        except Exception as e:
            print(f"Regression Error: {e}")
            return {
                "alpha": 0.0,
                "beta": 0.0,
                "r_squared": 0.0,
                "status": f"error: {str(e)}"
            }
