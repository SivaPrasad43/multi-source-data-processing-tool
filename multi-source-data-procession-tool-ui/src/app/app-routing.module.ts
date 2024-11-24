import { NgModule } from '@angular/core';
import { RouterModule, Routes } from '@angular/router';
import { InputConfigComponent } from './components/input-config/input-config.component';
import { OutputConfigComponent } from './components/output-config/output-config.component';

const routes: Routes = [
  {path: '', pathMatch: 'full', redirectTo: 'input-config'},
  {path: 'input-config', component: InputConfigComponent},
  {path: 'output-config', component: OutputConfigComponent},
];

@NgModule({
  imports: [RouterModule.forRoot(routes)],
  exports: [RouterModule]
})
export class AppRoutingModule { }
