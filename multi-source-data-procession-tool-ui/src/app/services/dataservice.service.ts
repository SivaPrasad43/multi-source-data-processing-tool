import { Injectable } from '@angular/core';
import { HttpClient, HttpErrorResponse, HttpHeaders } from '@angular/common/http';
import { Observable, throwError } from 'rxjs';
import { catchError, map, retry } from 'rxjs/operators';

// Interface for your data model
export interface Item {
  id: number;
  name: string;
  description?: string;
  // Add other properties as needed
}

@Injectable({
  providedIn: 'root'
})

export class DataserviceService {

  private gatewayUrl = 'http://localhost:8000'; // Replace with your API endpoint
  
  private httpOptions = {
    headers: new HttpHeaders({
      'Content-Type': 'application/json',
      // Add any other required headers
    })
  };

  constructor(private http: HttpClient) { }

  createConfiguration(confType:string,payload:any): Observable<any> {
  return this.http.post<any>(this.gatewayUrl+"/createConfiguration?configType="+confType,payload,this.httpOptions);
}

loadConfiguration(configType:string): Observable<any> {
  return this.http.get<any>(this.gatewayUrl+"/loadConfiguration?configType="+configType,this.httpOptions);
}

deployConfiguration(confType:string,payload:any): Observable<any> {
  return this.http.post<any>(this.gatewayUrl+"/deployConfiguration?configType="+confType,payload,this.httpOptions);
}

  // Error handling
  private handleError(error: HttpErrorResponse) {
    let errorMessage = 'An unknown error occurred!';
    
    if (error.error instanceof ErrorEvent) {
      // Client-side error
      errorMessage = `Error: ${error.error.message}`;
    } else {
      // Server-side error
      errorMessage = `Error Code: ${error.status}\nMessage: ${error.message}`;
    }
    
    console.error(errorMessage);
    return throwError(() => new Error(errorMessage));
  }

  // Example of custom data transformation
  private transformResponse<T>(response: any): T {
    // Add any data transformation logic here
    return response as T;
  }
}












