import { Injectable } from '@angular/core';
import { HttpClient, HttpHeaders } from '@angular/common/http';
import { Observable } from 'rxjs';

const httpOptions = {
  headers: new HttpHeaders({
    'Content-Type': 'application/json'
  })
};

@Injectable({
  providedIn: 'root',
})
export class DataserviceService {

  private gatewayUrl = 'http://localhost:8000';

  constructor(private http: HttpClient) { }

  createConfiguration(payload:any): Observable<any> {
    return this.http.post<any>(this.gatewayUrl+"/createConfiguration",payload,httpOptions);
  }
  
  readConfigyrationFile(): Observable<any> {
    return this.http.get<any>(this.gatewayUrl+"/readConfigyrationFile",httpOptions);
  }

  deployConfigData(payload:any): Observable<any> {
    return this.http.post<any>(this.gatewayUrl+"/deployConfigData",payload,httpOptions);
  }

}

