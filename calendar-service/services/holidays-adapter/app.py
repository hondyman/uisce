#!/usr/bin/env python3
"""
Holidays PyPI Adapter Microservice
Exposes holidays package as a REST API for the Semantic Engine
API Port: 8001
"""

from flask import Flask, jsonify, request
import holidays
import logging
from datetime import datetime

app = Flask(__name__)
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

# ============================================================================
# Holidays Library Registry (Maps region codes to country classes)
# ============================================================================

# Map of supported countries
SUPPORTED_COUNTRIES = {
    'US': 'US',
    'CA': 'CA',  # Canada
    'GB': 'GB',  # United Kingdom
    'FR': 'FR',  # France
    'DE': 'DE',  # Germany
    'IT': 'IT',  # Italy
    'ES': 'ES',  # Spain
    'JP': 'JP',  # Japan
    'CN': 'CN',  # China
    'IN': 'IN',  # India
    'BR': 'BR',  # Brazil
    'MX': 'MX',  # Mexico
    'AU': 'AU',  # Australia
}

# US State codes
US_STATES = ['AL', 'AK', 'AZ', 'AR', 'CA', 'CO', 'CT', 'DE', 'FL', 'GA',
             'HI', 'ID', 'IL', 'IN', 'IA', 'KS', 'KY', 'LA', 'ME', 'MD',
             'MA', 'MI', 'MN', 'MS', 'MO', 'MT', 'NE', 'NV', 'NH', 'NJ',
             'NM', 'NY', 'NC', 'ND', 'OH', 'OK', 'OR', 'PA', 'RI', 'SC',
             'SD', 'TN', 'TX', 'UT', 'VT', 'VA', 'WA', 'WV', 'WI', 'WY']

# ============================================================================
# API Endpoints
# ============================================================================

@app.route('/health', methods=['GET'])
def health():
    """Health check endpoint"""
    return jsonify({"status": "healthy", "service": "holidays-adapter"}), 200


@app.route('/holidays', methods=['GET'])
def get_holidays():
    """
    GET /holidays?region=US&year=2026
    GET /holidays?region=US&state=CA&year=2026
    Returns holidays for a specific region/country and year
    """
    region = request.args.get('region', '').upper()
    state = request.args.get('state', '').upper()
    year = request.args.get('year', type=int)

    if not region or not year:
        return jsonify({"error": "Missing region or year parameter"}), 400

    if region not in SUPPORTED_COUNTRIES:
        return jsonify({"error": f"Unsupported region: {region}"}), 400

    try:
        # Handle US state-specific holidays
        if region == 'US' and state:
            if state not in US_STATES:
                return jsonify({"error": f"Unsupported US state: {state}"}), 400
            holiday_obj = holidays.US(state=state, years=year)
        else:
            holiday_obj = holidays.country_holidays(SUPPORTED_COUNTRIES[region], years=year)

        holidays_list = [
            {
                "date": str(date),
                "name": name
            }
            for date, name in sorted(holiday_obj.items())
        ]

        result = {
            "region": region,
            "state": state if state else None,
            "year": year,
            "holidays": holidays_list
        }

        logger.info(f"Fetched {len(holidays_list)} holidays for {region}{f'/{state}' if state else ''}/{year}")
        return jsonify(result), 200

    except Exception as e:
        logger.error(f"Error fetching holidays: {e}")
        return jsonify({"error": str(e)}), 500


@app.route('/supported-regions', methods=['GET'])
def supported_regions():
    """Returns list of supported regions"""
    return jsonify({
        "countries": list(SUPPORTED_COUNTRIES.keys()),
        "us_states": US_STATES
    }), 200


@app.route('/is-holiday', methods=['GET'])
def is_holiday():
    """
    GET /is-holiday?region=US&date=2026-07-04
    GET /is-holiday?region=US&state=CA&date=2026-07-04
    Check if a specific date is a holiday
    """
    region = request.args.get('region', '').upper()
    state = request.args.get('state', '').upper()
    date_str = request.args.get('date', '')

    if not region or not date_str:
        return jsonify({"error": "Missing region or date parameter"}), 400

    if region not in SUPPORTED_COUNTRIES:
        return jsonify({"error": f"Unsupported region: {region}"}), 400

    try:
        date = datetime.strptime(date_str, "%Y-%m-%d").date()

        # Handle US state-specific holidays
        if region == 'US' and state:
            if state not in US_STATES:
                return jsonify({"error": f"Unsupported US state: {state}"}), 400
            holiday_obj = holidays.US(state=state, years=date.year)
        else:
            holiday_obj = holidays.country_holidays(SUPPORTED_COUNTRIES[region], years=date.year)

        holiday_name = holiday_obj.get(date)
        is_holiday_flag = date in holiday_obj

        return jsonify({
            "region": region,
            "state": state if state else None,
            "date": date_str,
            "is_holiday": is_holiday_flag,
            "holiday_name": holiday_name
        }), 200

    except ValueError as e:
        return jsonify({"error": f"Invalid date format: {str(e)}"}), 400
    except Exception as e:
        logger.error(f"Error checking holiday: {e}")
        return jsonify({"error": str(e)}), 500


if __name__ == '__main__':
    logger.info("Starting Holidays PyPI Adapter on port 8001")
    app.run(host='0.0.0.0', port=8001, debug=False)
