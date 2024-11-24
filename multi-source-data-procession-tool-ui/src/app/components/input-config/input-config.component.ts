import { Component, OnInit } from '@angular/core';

import { DataserviceService } from 'src/app/services/dataservice.service';

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
  templateUrl: './input-config.component.html',
  styleUrls: ['./input-config.component.css']
})
export class InputConfigComponent implements OnInit {

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

  configData:{
    ID:number,
    NAME:string,
    TYPE:string
    DATA: any[]
  }={
    ID:0,
    NAME: "",
    TYPE: "",
    DATA: []
  }

  payload:any[]=[]

  tableData:{
    ID:number,
    NAME:string,
    DATA:any[]
  }[]=[]

  confId:number=0

  constructor(
    private dataservice:DataserviceService
  ) { }

  ngOnInit(): void {
    this.loadInputConfig()
  }

  loadInputConfig(){
    this.dataservice.loadConfiguration("sourceConfig").subscribe({
      next: (response) => {
        console.log(JSON.parse(response.data));
        const data = JSON.parse(response.data)
        console.log(response.data.SourceData)
        data.SourceData.forEach((item:any) => {
          this.tableData.push({
            ID: item.Source,
            NAME: item.NAME,
            DATA: item.TYPEOF
          })
        })
        this.confId = this.tableData[this.tableData.length-1].ID
      },
      error: (error) => {
        console.error(error);
      },
      complete: () => {
        console.log('Configuration creation completed');
      }
    })
  }

  handleFileInput(event: Event): void {
    const input = event.target as HTMLInputElement;
    if (input.files && input.files.length > 0) {
      const file = input.files[0];
      console.log(file)
      if (file.type !== 'text/csv' && file.type !== 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet') {
        alert('Please select a CSV or Excel file');
        return;
      }
      const reader = new FileReader();
      reader.onload = (e) => {
        this.fileData.FILE_DATA = btoa(e.target?.result as string);
        console.log('Selected file:', file.name);
      };
      reader.readAsBinaryString(file);
    }
  }

  addInput(){
    switch(this.configData.TYPE){
      case "DB":
        this.dbData.TYPE = this.configData.TYPE
        this.payload.push(this.dbData)
        break;
      case "FILE":
        this.fileData.TYPE = this.configData.TYPE
        this.payload.push(this.fileData)
        break;
      case "HTTP":
        this.httpData.TYPE = this.configData.TYPE
        this.payload.push(this.httpData)
        break;
      case "KAFKA":
        this.cloudData.TYPE = this.configData.TYPE
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
    console.log(this.payload)

    this.tableData.push({
      ID: this.confId+1,
      NAME: this.configData.NAME,
      DATA: this.payload
    })

    this.confId = this.confId+1


  }

  syncInputs(){
    let payload:any[] = []
    this.tableData.forEach((item) => {
      payload.push({
        Source: item.ID,
        NAME: item.NAME,
        TYPEOF: item.DATA
      })
    })
    this.dataservice.createConfiguration("sourceConfig",{
      SourceData: payload
    }).subscribe({
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
  
  deployInput(payload:any){
    this.dataservice.deployConfiguration("sourceConfig",payload).subscribe({
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
