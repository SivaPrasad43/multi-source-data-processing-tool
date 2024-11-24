import { NgModule } from '@angular/core';
import { BrowserModule } from '@angular/platform-browser';
import { FormsModule } from '@angular/forms';
import { CommonModule } from '@angular/common';
import { HttpClientModule } from '@angular/common/http';

import { AppRoutingModule } from './app-routing.module';
import { AppComponent } from './app.component';
import { SidebarComponent } from './components/sidebar/sidebar.component';
import { InputConfigComponent } from './components/input-config/input-config.component';
import { OutputConfigComponent } from './components/output-config/output-config.component';

import { DataserviceService } from './services/dataservice.service';

@NgModule({
  declarations: [
    AppComponent,
    SidebarComponent,
    InputConfigComponent,
    OutputConfigComponent
  ],
  imports: [
    BrowserModule,
    AppRoutingModule,
    FormsModule,
    CommonModule,
    HttpClientModule
  ],
  providers: [
    DataserviceService,
    
  ],
  bootstrap: [AppComponent]
})
export class AppModule { }
