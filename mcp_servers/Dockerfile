FROM docker.io/library/python:3.12-slim                                                         
                                                                                           
ENV PYTHONDONTWRITEBYTECODE=1 PYTHONUNBUFFERED=1                                                        
                                                                                
WORKDIR /app                                                                  
                                                                                
# Install dependencies first for layer caching                                
COPY mcp_servers/requirements.txt .                                           
RUN pip install --no-cache-dir -r requirements.txt                            
                                                                                
# Copy server sources and ML model artifacts                                  
COPY mcp_servers/ .                                                           
                                                                                
EXPOSE 9000                                                                   
                                                                                
CMD ["python", "main.py"]    