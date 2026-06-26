import yaml

def clean_compose(file_path):
    with open(file_path, 'r') as f:
        data = yaml.safe_load(f)
    
    if 'services' in data:
        for service_name, config in data['services'].items():
            if 'networks' in config:
                # Deduplicate and ensure it's a list with semlayer-net
                if isinstance(config['networks'], list):
                    config['networks'] = list(set(config['networks']))
                    if 'semlayer-net' not in config['networks']:
                        config['networks'].append('semlayer-net')
                else:
                    config['networks'] = ['semlayer-net']
            else:
                config['networks'] = ['semlayer-net']
                
    with open(file_path, 'w') as f:
        yaml.dump(data, f, default_flow_style=False, sort_keys=False)

if __name__ == "__main__":
    clean_compose('docker-compose.yml')
    clean_compose('docker-compose.local.yml')
    clean_compose('docker-compose.override.yml')
