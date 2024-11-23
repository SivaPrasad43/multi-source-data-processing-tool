import { Routes } from '@angular/router';
import { InputConfigComponent } from './input-config/input-config.component';
import { DashboardComponent } from './dashboard/dashboard.component';
import { OutputConfigComponent } from './output-config/output-config.component';

export const routes: Routes = [
  {
    path: '',
    pathMatch: 'full',
    redirectTo: 'input-config'
  },
  {
    path: 'input-config',
    component: InputConfigComponent
  },
  {
    path: 'output-config',
    component: OutputConfigComponent
  }
];

