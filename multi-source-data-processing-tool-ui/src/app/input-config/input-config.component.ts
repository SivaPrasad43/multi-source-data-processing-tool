import { Component } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { CommonModule } from '@angular/common';


import { DataserviceService } from '../dataservice.service';
import {HttpClient, HttpHandler} from '@angular/common/http';
import { HttpClientModule } from '@angular/common/http';

interface DB_CONF {
  TYPE: string;
  DB_TYPE: string;
  DB_HOST: string;
  DB_PORT: number;
  DB_USER: string;
  DB_PASSWORD: string;
  DB_NAME: string;
  DURATION_TIME: number;
  DURATION_TYPE:string
}

interface FILE_CONF {
  TYPE: string;
  FILE_TYPE: string;
  FILE_DATA: any;
  DURATION_TIME: number;
  DURATION_TYPE:string
}

interface CLOUD_CONF {
  TYPE: string;
  IP: string,
  PORT:  string,
  TOPIC: string
}

interface HTTP_CONF {
  TYPE: string;
  URL: string,
  DURATION_TIME: number;
  DURATION_TYPE:string
} 

@Component({
  selector: 'app-input-config',
  standalone: true,
  imports: [FormsModule,CommonModule],
  providers:[DataserviceService],
  templateUrl: './input-config.component.html',
  styleUrl: './input-config.component.css'
})


export class InputConfigComponent {
  

  selectedInputType:string=""
  
  dbData:DB_CONF = {
    TYPE: "",
    DB_TYPE: "",
    DB_HOST: "",
    DB_PORT: 0,
    DB_USER: "",
    DB_PASSWORD: "",
    DB_NAME: "",
    DURATION_TIME: 0,
    DURATION_TYPE:"Minute"
  }

  fileData:FILE_CONF = {
    TYPE: "",
    FILE_TYPE: "",
    FILE_DATA:"",
    DURATION_TIME: 0,
    DURATION_TYPE:"Minute"
  }

  cloudData:CLOUD_CONF = {
    TYPE: "",
    IP: "",
    PORT: "",
    TOPIC: ""
  }

  httpData:HTTP_CONF = {
    TYPE: "",
    URL: "",
    DURATION_TIME: 0,
    DURATION_TYPE:"Minute"
  }

  payload:any[]=[]


  constructor(private dataservice:DataserviceService) { }


  handleFileInput(event: Event): void {
    const input = event.target as HTMLInputElement;
    if (input.files && input.files.length > 0) {
      this.fileData.FILE_DATA = input.files[0];
      console.log('Selected file:', input.files[0].name);
    }
  }

  addInput(){
    switch(this.selectedInputType){
      case "Database":
        this.dbData.TYPE = this.selectedInputType
        this.payload.push(this.dbData)
        break;
      case "File":
        this.fileData.TYPE = this.selectedInputType
        this.payload.push(this.fileData)
        break;
      case "Http":
        this.httpData.TYPE = this.selectedInputType
        this.payload.push(this.httpData)
        break;
      case "Kafka":
        this.cloudData.TYPE = this.selectedInputType
        this.payload.push(this.cloudData)
        break;
      default:
        break
    }
  }

  removeFile(index: number) {
    this.payload.splice(index, 1);
  }

  createInputConfig(){
    this.dataservice.createConfiguration(this.payload).subscribe({
      next: (response) => {
        console.log(response);
      },
      error: (error) => {
        console.error(error);
      },
      complete: () => {
        console.log('Configuration creation completed');
      }
    })
  }
  

}
