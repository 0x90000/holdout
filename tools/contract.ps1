$ErrorActionPreference = 'Stop'
python (Join-Path $PSScriptRoot 'contract.py')
exit $LASTEXITCODE
